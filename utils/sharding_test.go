package utils

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/go-logr/logr/testr"
	corev1 "k8s.io/api/core/v1"
)

func Test_getEndpointChanges(t *testing.T) {
	testCases := []struct {
		field        string
		anotherfield string
	}{
		{
			field: "todo",
		},
		{
			field:        "todo",
			anotherfield: "todo",
		},
	}

	// u := &unstructured.Unstructured{
	// 	Object: map[string]interface{}{
	// 		"apiVersion": "cr.bar.com/v1",
	// 		"kind":       "Foo",
	// 		"spec":       map[string]interface{}{"field": 1},
	// 		"metadata": map[string]interface{}{
	// 			"name": "test-1",
	// 			"annotations": map[string]interface{}{
	// 				"foo": "bar",
	// 			},
	// 		},
	// 	},
	// }

	// a1 := map[string]interface{}{
	// 	"ip":       "10.0.131.187",
	// 	"nodeName": "worker-000026",
	// 	"targetRef": map[string]string{
	// 		"kind":      "Pod",
	// 		"name":      "starboard-exporter-765756bd69-vzx49",
	// 		"namespace": "giantswarm",
	// 	},
	// }

	// addresses := make([]map[string]interface{})
	// addresses = append(addresses)

	// ep := &unstructured.Unstructured{
	// 	Object: map[string]interface{}{
	// 		"apiVersion": "v1",
	// 		"kind":       "Endpoints",
	// 		"metadata": map[string]interface{}{
	// 			"name":      "starboard-exporter",
	// 			"namespace": "giantswarm",
	// 		},
	// 		// "subsets": map[string][]map[string]interface{}{
	// 		"subsets": map[string]interface{}{
	// 			"addresses": []map[string]interface{}{
	// 				{
	// 					"ip":       "10.0.131.187",
	// 					"nodeName": "worker-000026",
	// 					"targetRef": map[string]string{
	// 						"kind":      "Pod",
	// 						"name":      "starboard-exporter-765756bd69-vzx49",
	// 						"namespace": "giantswarm",
	// 					},
	// 				},
	// 			},
	// 			"ports": []map[string]interface{}{
	// 				{
	// 					"name":     "metrics",
	// 					"port":     "8080",
	// 					"protocol": "TCP",
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	epp := &corev1.Endpoints{}
	subsets := corev1.EndpointSubset{
		Addresses: []corev1.EndpointAddress{
			{
				IP: "1.2.3.4",
				// NodeName: "worker-000026",
			},
		},
	}

	epp.Subsets = []corev1.EndpointSubset{subsets}

	log := testr.New(t)

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			log.Info("test")
			log.Info(fmt.Sprintf("%v", epp))
			added, kept, removed, ok := getEndpointChanges(epp, nil, log)
			t.Logf("case %s: added: %s, kept: %s, removed: %s\n", testCase, added, kept, removed)

			if !ok {
				t.Fatalf("case %s: added: %s, kept: %s, removed: %s\n", testCase, added, kept, removed)
			}
		})
	}
}

//	{
//   - ip: 10.0.133.125
// 	nodeName: worker-000025
// 	targetRef:
// 	  kind: Pod
// 	  name: starboard-exporter-765756bd69-r795x
// 	  namespace: giantswarm
// 	  resourceVersion: "694068745"
// 	  uid: 02758daa-e56a-499f-af8a-fb795890700f
// 	  }
