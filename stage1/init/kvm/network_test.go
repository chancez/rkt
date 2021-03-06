// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kvm

import (
	"net"
	"testing"
)

type testNetDescriber struct {
	hostIP  net.IP
	guestIP net.IP
	mask    net.IP
	ifName  string
	ipMasq  bool
}

func (t testNetDescriber) HostIP() net.IP  { return t.hostIP }
func (t testNetDescriber) GuestIP() net.IP { return t.guestIP }
func (t testNetDescriber) Mask() net.IP    { return t.mask }
func (t testNetDescriber) IfName() string  { return t.ifName }
func (t testNetDescriber) IPMasq() bool    { return t.ipMasq }

func TestGetKVMNetArgs(t *testing.T) {
	tests := []struct {
		netDescriptions []netDescriber
		expectedLkvm    []string
		expectedKernel  []string
	}{
		{ // without Masquerading - not gw passed to kernel
			netDescriptions: []netDescriber{
				testNetDescriber{
					net.ParseIP("1.1.1.1"),
					net.ParseIP("2.2.2.2"),
					net.ParseIP("255.255.255.0"),
					"fooInt",
					false,
				},
			},
			expectedLkvm:   []string{"--network", "mode=tap,tapif=fooInt,host_ip=1.1.1.1,guest_ip=2.2.2.2"},
			expectedKernel: []string{"ip=2.2.2.2:::255.255.255.0::eth0:::"},
		},
		{ // extra gw passed to kernel on (thrid position)
			netDescriptions: []netDescriber{
				testNetDescriber{
					net.ParseIP("1.1.1.1"),
					net.ParseIP("2.2.2.2"),
					net.ParseIP("255.255.255.0"),
					"barInt", true},
			},
			expectedLkvm:   []string{"--network", "mode=tap,tapif=barInt,host_ip=1.1.1.1,guest_ip=2.2.2.2"},
			expectedKernel: []string{"ip=2.2.2.2::1.1.1.1:255.255.255.0::eth0:::"},
		},
		{ // two networks
			netDescriptions: []netDescriber{
				testNetDescriber{
					net.ParseIP("1.1.1.1"),
					net.ParseIP("2.2.2.2"),
					net.ParseIP("255.255.255.0"),
					"fooInt",
					false,
				},
				testNetDescriber{
					net.ParseIP("1.1.1.1"),
					net.ParseIP("2.2.2.2"),
					net.ParseIP("255.255.255.0"),
					"barInt", true},
			},
			expectedLkvm: []string{
				"--network", "mode=tap,tapif=fooInt,host_ip=1.1.1.1,guest_ip=2.2.2.2",
				"--network", "mode=tap,tapif=barInt,host_ip=1.1.1.1,guest_ip=2.2.2.2",
			},
			expectedKernel: []string{
				"ip=2.2.2.2:::255.255.255.0::eth0:::",
				"ip=2.2.2.2::1.1.1.1:255.255.255.0::eth1:::",
			},
		},
	}

	for i, tt := range tests {
		gotLkvm, gotKernel, err := GetKVMNetArgs(tt.netDescriptions)
		if err != nil {
			t.Errorf("got error: %s", err)
		}
		if len(gotLkvm) != len(tt.expectedLkvm) {
			t.Errorf("#%d: expected lkvm %v elements got %v", i, len(tt.expectedLkvm), len(gotLkvm))
		} else {
			for iarg, argExpected := range tt.expectedLkvm {
				if gotLkvm[iarg] != argExpected {
					t.Errorf("#%d: lkvm arg %d expected `%v` got `%v`", i, iarg, argExpected, gotLkvm[iarg])
				}
			}
		}
		if len(gotKernel) != len(tt.expectedKernel) {
			t.Errorf("#%d: expected kernel %v elements got %v", i, len(tt.expectedKernel), len(gotKernel))
		} else {
			for iarg, argExpected := range tt.expectedKernel {
				if gotKernel[iarg] != argExpected {
					t.Errorf("#%d: kernel arg %d expected `%v` got `%v`", i, iarg, argExpected, gotKernel[iarg])
				}
			}
		}
	}
}
