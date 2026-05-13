package utils

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	discoveryv1 "k8s.io/api/discovery/v1"
)

func Test_endpointSlicesToIPv4Set(t *testing.T) {
	ready := true
	notReady := false

	testCases := []struct {
		name     string
		slices   []*discoveryv1.EndpointSlice
		expected []string
	}{
		{
			name: "single ready ipv4 endpoint",
			slices: []*discoveryv1.EndpointSlice{
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
			slices: []*discoveryv1.EndpointSlice{
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
			slices: []*discoveryv1.EndpointSlice{
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
			slices: []*discoveryv1.EndpointSlice{
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
			name:     "empty input returns empty set",
			slices:   nil,
			expected: []string{},
		},
		{
			name: "nil slices are ignored",
			slices: []*discoveryv1.EndpointSlice{
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
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ipv4Set := endpointSlicesToIPv4Set(tc.slices)
			t.Logf("case %v: ipv4Set: %v\n", tc, ipv4Set)

			// Convert the ipv4Set to a slice for comparison.
			ips := make([]string, 0, len(ipv4Set))
			for ip := range ipv4Set {
				ips = append(ips, ip)
			}
			compareStringFunc := func(a, b string) bool { return a < b }

			// Check IPs contain the expected items, ignoring order.
			assert.Assert(t, cmp.Equal(tc.expected, ips, cmpopts.EquateEmpty(), cmpopts.SortSlices(compareStringFunc)), "test case %v failed.", tc.name)
		})
	}
}
