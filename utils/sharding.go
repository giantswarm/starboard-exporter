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

func (r *ShardHelper) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.ring.GetMembers())
}

func (r *ShardHelper) GetShardOwner(input string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ring.LocateKey([]byte(input)).String()
}

func (r *ShardHelper) ShouldOwn(input string) bool {
	return r.GetShardOwner(input) == r.PodIP
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
			updateRingFromEndpoints(peerRing, obj, ringConfig, log)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			updateRingFromEndpoints(peerRing, newObj, ringConfig, log)
		},
		// TODO: Delete handler
	}

	informer.AddEventHandler(handlers)
	return informer
}

func updateRingFromEndpoints(ring *ShardHelper, obj interface{}, ringConfig consistent.Config, log logr.Logger) {
	ep, err := toEndpoint(obj)
	if err != nil {
		log.Error(err, "could not convert obj to Endpoints")
		return
	}

	// TODO: This should modify add/remove members instead of re-creating the whole ring
	ring.mu.Lock()
	defer ring.mu.Unlock()
	ring.ring = consistent.New(nil, ringConfig)

	fmt.Println("current IPs:")
	for _, subset := range ep.Subsets {
		for _, ip := range subset.Addresses {
			fmt.Println(ip.IP)
			ring.ring.Add(peer(ip.IP))
		}
	}

	log.Info(fmt.Sprintf("synchronized peer ring with %d peers", len(ring.ring.GetMembers())))
}

func toEndpoint(obj interface{}) (*corev1.Endpoints, error) {
	ep := &corev1.Endpoints{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), ep)
	if err != nil {
		fmt.Println("could not convert obj to Endpoints")
		fmt.Print(err)
		return ep, err
	}
	return ep, nil
}
