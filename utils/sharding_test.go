package utils

import (
	"strconv"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_getEndpointChanges(t *testing.T) {
	testCases := []struct {
		name            string
		current         *corev1.Endpoints
		previous        *corev1.Endpoints
		expectedAdded   []string
		expectedKept    []string
		expectedRemoved []string
	}{
		{
			name: "add one new endpoint with no previous state",
			current: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.2.3.4",
							},
						},
					},
				},
			},
			expectedAdded:   []string{"1.2.3.4"},
			expectedKept:    []string{},
			expectedRemoved: []string{},
		},
		{
			name: "add one new endpoint to one previous endpoint",
			current: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.2.3.4",
							},
							{
								IP: "5.6.7.8",
							},
						},
					},
				},
			},
			previous: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.2.3.4",
							},
						},
					},
				},
			},
			expectedAdded:   []string{"5.6.7.8"},
			expectedKept:    []string{"1.2.3.4"},
			expectedRemoved: []string{},
		},
		{
			name: "add multiple new endpoints to two previous endpoints",
			current: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
							{
								IP: "8.8.4.4",
							},
							{
								IP: "1.2.3.4",
							},
							{
								IP: "5.6.7.8",
							},
						},
					},
				},
			},
			previous: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "1.2.3.4",
							},
							{
								IP: "5.6.7.8",
							},
						},
					},
				},
			},
			expectedAdded:   []string{"8.8.4.4", "8.8.8.8"},
			expectedKept:    []string{"1.2.3.4", "5.6.7.8"},
			expectedRemoved: []string{},
		},
		{
			name: "remove multiple endpoints",
			current: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
							{
								IP: "1.2.3.4",
							},
						},
					},
				},
			},
			previous: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
							{
								IP: "8.8.4.4",
							},
							{
								IP: "1.2.3.4",
							},
							{
								IP: "5.6.7.8",
							},
						},
					},
				},
			},
			expectedAdded:   []string{},
			expectedKept:    []string{"1.2.3.4", "8.8.8.8"},
			expectedRemoved: []string{"5.6.7.8", "8.8.4.4"},
		},
		{
			name: "add and remove endpoints in one udpate",
			current: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "8.8.4.4",
							},
							{
								IP: "1.2.3.4",
							},
							{
								IP: "5.6.7.8",
							},
						},
					},
				},
			},
			previous: &corev1.Endpoints{
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "8.8.8.8",
							},
							{
								IP: "1.2.3.4",
							},
						},
					},
				},
			},
			expectedAdded:   []string{"5.6.7.8", "8.8.4.4"},
			expectedKept:    []string{"1.2.3.4"},
			expectedRemoved: []string{"8.8.8.8"},
		},
	}

	// Logger to pass to helper functions. Wraps testing.T.
	log := testr.New(t)

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var previous *corev1.Endpoints
			{
				previous = nil
				if tc.previous != nil {
					previous = tc.previous
				}
			}

			// Calculate endpoint updates.
			added, kept, removed, ok := getEndpointChanges(tc.current, previous, log)

			t.Logf("case %v: added: %v, kept: %v, removed: %v\n", tc, added, kept, removed)

			if !ok {
				t.Fatalf("unable to parse endpoint changes for case %v: added: %s, kept: %s, removed: %s\n", tc, added, kept, removed)
			}

			compareStringFunc := func(a, b string) bool { return a < b }

			// Check added, kept, and removed contain the expected items, ignoring order.
			assert.Assert(t, cmp.Equal(tc.expectedAdded, added, cmpopts.EquateEmpty(), cmpopts.SortSlices(compareStringFunc)), "test case %v failed.", tc.name)
			assert.Assert(t, cmp.Equal(tc.expectedKept, kept, cmpopts.EquateEmpty(), cmpopts.SortSlices(compareStringFunc)), "test case %v failed.", tc.name)
			assert.Assert(t, cmp.Equal(tc.expectedRemoved, removed, cmpopts.EquateEmpty(), cmpopts.SortSlices(compareStringFunc)), "test case %v failed.", tc.name)
		})
	}
}
