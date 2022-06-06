package utils

import (
	"fmt"
	"os"
	"sync"

	"github.com/buraksezer/consistent"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ShardHelper struct {
	PodIP            string
	ServiceName      string
	ServiceNamespace string
	mu               *sync.RWMutex
	ring             *consistent.Consistent
}

// Returns the number of members/peers currently in the hash ring.
func (r *ShardHelper) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.ring.GetMembers())
}

// Returns the name (IP) of the shard which should own a provided object name.
func (r *ShardHelper) GetShardOwner(input string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ring.LocateKey([]byte(input)).String()
}

// Returns whether the current shard should own the object with the provided name.
func (r *ShardHelper) ShouldOwn(input string) bool {
	return r.GetShardOwner(input) == r.PodIP
}

// SetMembers accepts a map where the keys are member IPs and uses those IPs as the members for sharding.
func (r *ShardHelper) SetMembers(newMembers map[string]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add new members. Add returns immediately if the member already exists.
	for newMember := range newMembers {
		r.ring.Add(peer(newMember))
	}

	// Remove members which don't exist anymore.
	for _, oldMember := range r.ring.GetMembers() {
		if _, ok := newMembers[oldMember.String()]; !ok {
			r.ring.Remove(oldMember.String())
		}
	}
}

// SetMembersFromLists is a wrapper around SetMember which accepts slices instead of a map.
func (r *ShardHelper) SetMembersFromLists(lists ...[]string) {
	members := make(map[string]struct{})
	for _, l := range lists {
		for _, m := range l {
			members[m] = struct{}{}
		}
	}
	r.SetMembers(members)
}

// Helper type for members of peer ring.
type peer string

func (p peer) String() string {
	return string(p)
}

func BuildPeerHashRing(consistentCfg consistent.Config, podIP string, serviceName string, serviceNamespace string) *ShardHelper {
	consistentHashRing := consistent.New(nil, consistentCfg)
	mutex := sync.RWMutex{}
	return &ShardHelper{
		PodIP:            podIP,
		ServiceName:      serviceName,
		ServiceNamespace: serviceNamespace,
		mu:               &mutex,
		ring:             consistentHashRing,
	}
}

func BuildPeerInformer(stopper chan struct{}, peerRing *ShardHelper, ringConfig consistent.Config, log logr.Logger) cache.SharedIndexInformer {

	dc, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		log.Error(err, "unable to set up informer")
		os.Exit(1)
	}

	listOptionsFunc := dynamicinformer.TweakListOptionsFunc(func(options *metav1.ListOptions) {
		options.FieldSelector = "metadata.name=" + peerRing.ServiceName
	})

	// Use our namespace and expected endpoints name in our future informer.
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, peerRing.ServiceNamespace, listOptionsFunc)

	// Construct an informer for the endpoints.
	informer := factory.ForResource(schema.GroupVersionResource{
		Group: "", Version: "v1", Resource: "endpoints"}).Informer()

	// Set handlers for new/updated endpoints.
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			updateEndpoints(obj, nil, peerRing, log)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// In the future, we might need to re-queue objects which belong to deleted peers.
			// When scaling down, it is possible that metrics will be double reported for up to the reconciliation period.
			// For now, we'll just set the desired peers.
			updateEndpoints(newObj, oldObj, peerRing, log)
		},
		// We can add a delete handler here. Not sure yet what it should do.
	}

	informer.AddEventHandler(handlers)
	return informer
}

func updateEndpoints(currentObj interface{}, previousObj interface{}, ring *ShardHelper, log logr.Logger) {
	added, kept, _, ok := getEndpointChanges(currentObj, previousObj, log)
	if !ok {
		return
	}
	ring.SetMembersFromLists(added, kept)
	log.Info(fmt.Sprintf("updated peer list with %d endpoints", len(added)+len(kept)))
}

// getEndpointChanges takes a current and optional previous object and returns the added, kept, and removed items, plus a success boolean.
func getEndpointChanges(currentObj interface{}, previousObj interface{}, log logr.Logger) ([]string, []string, []string, bool) {
	current, err := toEndpoint(currentObj, log)
	if err != nil {
		log.Error(err, "could not convert obj to Endpoints")
		return nil, nil, nil, false
	}

	currentEndpoints := []string{}                   // Stores current endpoints to return directly if we don't have a previous state.
	currentEndpointsMap := make(map[string]struct{}) // Stores the endpoints as a map for quicker comparisons to previous state.

	// Store all the current endpoints for us to use later.
	for _, subset := range current.Subsets {
		for _, ip := range subset.Addresses {
			// We add to both the list and the map. This wastes a little memory because we only ever need one or the other,
			// but it saves cycles to not loop over the endpoints multiple times. We don't expect tons of endpoints.
			currentEndpoints = append(currentEndpoints, ip.IP)
			currentEndpointsMap[ip.IP] = struct{}{}
		}
	}

	if previousObj == nil {
		// If there is no previous object, we're only adding new (initial) endpoints.
		// Just return the current endpoint list.
		return currentEndpoints, nil, nil, true
	}

	previous, err := toEndpoint(previousObj, log)
	if err != nil {
		log.Error(err, "could not convert obj to Endpoints")
		return nil, nil, nil, false
	}

	added := []string{}
	kept := []string{}
	removed := []string{}

	for _, subset := range previous.Subsets {
		for _, ip := range subset.Addresses {
			// Each address was either:
			// - added   (exists in current, not previous)
			// - kept    (exists in current and previous)
			// - removed (not in current, exists in previous)

			if _, found := currentEndpointsMap[ip.IP]; !found {
				// Endpoint has been removed in current state.
				removed = append(removed, ip.IP)
			} else {
				// Item existed before, so it has been "kept" and not "added".
				kept = append(kept, ip.IP)
				delete(currentEndpointsMap, ip.IP)
			}
		}
	}

	// Any remaining items in the added endpoints map were actually new. Return them as a list.
	for ip := range currentEndpointsMap {
		added = append(added, ip)
	}

	return added, kept, removed, true

}

func toEndpoint(obj interface{}, log logr.Logger) (*corev1.Endpoints, error) {
	ep := &corev1.Endpoints{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), ep)
	if err != nil {
		log.Error(err, "could not convert obj to Endpoints")
		return ep, err
	}
	return ep, nil
}
