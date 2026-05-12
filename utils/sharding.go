package utils

import (
	"fmt"
	"os"
	"sync"

	"github.com/buraksezer/consistent"
	"github.com/go-logr/logr"
	discoveryv1 "k8s.io/api/discovery/v1"
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
		options.LabelSelector = "kubernetes.io/service-name=" + peerRing.ServiceName
	})

	// Use our namespace and expected endpoints name in our future informer.
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, peerRing.ServiceNamespace, listOptionsFunc)

	// Construct an informer for the endpointslices.
	informer := factory.ForResource(schema.GroupVersionResource{
		Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}).Informer()

	// Set handlers for new/updated/deleted endpointslices.
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Might be valid to add members based on just the added EndpointSlice.
			// But to be safe we will update members based on all EndpointSlices.
			updateAllEndpoints(informer, peerRing, log)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// On update some of the endpoints may have been moved to a different EndpointSlice.
			// We have to set members based on all EndpointSlices.
			updateAllEndpoints(informer, peerRing, log)
		},
		DeleteFunc: func(obj interface{}) {
			// On delete some of the endpoints may exist in other EndpointSlices.
			// We have to set members based on all EndpointSlices.
			updateAllEndpoints(informer, peerRing, log)
		},
	}

	_, err = informer.AddEventHandler(handlers)
	if err != nil {
		log.Info(err.Error(), "error adding event handler to informer")
	}
	return informer
}

// updateAllEndpoints lists all EndpointSlices for the configured service and updates the ring members.
func updateAllEndpoints(informer cache.SharedIndexInformer, ring *ShardHelper, log logr.Logger) {
	// List all EndpointSlices for the service.
	// We use the informer store to list EndpointSlices to reduce load on the API server.
	list := informer.GetStore().List()

	// Collect unique IPs across all EndpointSlices.
	ipSet := make(map[string]struct{})
	for _, item := range list {
		eps, err := toEndpointSlice(item, log)
		if err != nil {
			log.Error(err, "could not convert item to EndpointSlice")
			continue
		}

		// Only consider IPv4 EndpointSlices
		if eps.AddressType != discoveryv1.AddressTypeIPv4 {
			continue
		}

		for _, ep := range eps.Endpoints {
			if ep.Conditions.Ready != nil && !*ep.Conditions.Ready {
				// Skip endpoints which are not ready.
				continue
			}
			for _, ip := range ep.Addresses {
				ipSet[ip] = struct{}{}
			}
		}
	}

	// Update ring members with the collected IPs.
	ring.SetMembers(ipSet)
	log.Info(fmt.Sprintf("updated peer list with %d endpoints (from EndpointSlices)", len(ipSet)))
}

func toEndpointSlice(obj interface{}, log logr.Logger) (*discoveryv1.EndpointSlice, error) {
	eps := &discoveryv1.EndpointSlice{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), eps)
	if err != nil {
		log.Error(err, "could not convert obj to EndpointSlice")
		return eps, err
	}
	return eps, nil
}
