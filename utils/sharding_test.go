package utils

import (
	"hash/fnv"
	"testing"

	"github.com/buraksezer/consistent"
	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	discoveryv1 "k8s.io/api/discovery/v1"
)

type testHasher struct{}

func (h testHasher) Sum64(data []byte) uint64 {
	hasher := fnv.New64a()
	hasher.Write(data)
	return hasher.Sum64()
}

func compareStringFn(a, b string) bool { return a < b }

func Test_updateAllEndpoints(t *testing.T) {
	ready := true
	notReady := false

	testCases := []struct {
		name     string
		previous []*discoveryv1.EndpointSlice
		current  []*discoveryv1.EndpointSlice
		expected []string
	}{
		{
			name: "single ready ipv4 endpoint",
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses:  []string{"1.2.3.4"},
							Conditions: discoveryv1.EndpointConditions{Ready: &ready},
						},
					},
				},
			},
			expected: []string{"1.2.3.4"},
		},
		{
			name: "nil ready is treated as ready",
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses:  []string{"2.2.2.2"},
							Conditions: discoveryv1.EndpointConditions{},
						},
					},
				},
			},
			expected: []string{"2.2.2.2"},
		},
		{
			name: "merges multiple slices and deduplicates addresses",
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"5.6.7.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"5.6.7.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"9.9.9.9"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"1.2.3.4", "5.6.7.8", "9.9.9.9"},
		},
		{
			name: "ignores not ready and ipv6 endpoints",
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &notReady}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"3.3.3.3"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv6,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"2001:db8::1"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"3.3.3.3"},
		},
		{
			name:     "empty slices result in empty endpoints",
			expected: []string{},
		},
		{
			name: "nil slices are ignored",
			current: []*discoveryv1.EndpointSlice{
				nil,
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"10.0.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"10.0.0.1"},
		},
		{
			name: "add multiple new endpoints to two previous endpoints",
			previous: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"5.6.7.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"8.8.8.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"8.8.4.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"5.6.7.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"8.8.4.4", "8.8.8.8", "1.2.3.4", "5.6.7.8"},
		},
		{
			name: "remove multiple endpoints",
			previous: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"8.8.4.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"8.8.8.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"5.6.7.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"8.8.8.8"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"1.2.3.4"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"1.2.3.4", "8.8.8.8"},
		},
		{
			name: "remove one and add one endpoint",
			previous: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"1.1.1.1"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"2.2.2.2"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			current: []*discoveryv1.EndpointSlice{
				{
					AddressType: discoveryv1.AddressTypeIPv4,
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"2.2.2.2"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
						{Addresses: []string{"3.3.3.3"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
					},
				},
			},
			expected: []string{"2.2.2.2", "3.3.3.3"},
		},
	}

	// Logger to pass to helper functions. Wraps testing.T.
	log := testr.New(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := consistent.Config{
				PartitionCount:    97,
				ReplicationFactor: 20,
				Load:              1.25,
				Hasher:            testHasher{},
			}
			testRing := BuildPeerHashRing(testCfg, "10.0.0.1", "starboard-exporter", "default")

			if tc.previous != nil {
				updateAllEndpoints(tc.previous, testRing, log)
			}

			updateAllEndpoints(tc.current, testRing, log)

			members := testRing.ring.GetMembers()
			ips := make([]string, 0, len(members))
			for _, member := range members {
				ips = append(ips, member.String())
			}

			// Check IPs contain the expected items, ignoring order.
			assert.Assert(t, cmp.Equal(tc.expected, ips, cmpopts.EquateEmpty(), cmpopts.SortSlices(compareStringFn)), "test case %v failed.", tc.name)
		})
	}
}
