/*
Copyright (c) 2025 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.
*/

package computeinstance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	publicv1 "github.com/osac-project/fulfillment-service/internal/api/osac/public/v1"
)

func Test_parseNetworkAttachmentFlag_WhenInputIsValidItShouldParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		input      string
		wantSubnet string
		wantSGs    []string
	}{
		{
			name:       "When value is bare subnet id it should use it as subnet",
			input:      "  sub-1  ",
			wantSubnet: "sub-1",
		},
		{
			name:       "When value is subnet=key form it should parse subnet",
			input:      "subnet=sub-2",
			wantSubnet: "sub-2",
		},
		{
			name:       "When value includes security_groups alias it should parse groups",
			input:      "subnet=a,security_groups=g1,g2",
			wantSubnet: "a",
			wantSGs:    []string{"g1", "g2"},
		},
		{
			name:       "When value lists security-groups after subnet it should parse groups",
			input:      "subnet=b,security-groups=x",
			wantSubnet: "b",
			wantSGs:    []string{"x"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseNetworkAttachmentFlag(tc.input)
			if err != nil {
				t.Fatalf("parseNetworkAttachmentFlag: %v", err)
			}
			if got.GetSubnet() != tc.wantSubnet {
				t.Fatalf("subnet: got %q want %q", got.GetSubnet(), tc.wantSubnet)
			}
			if diff := cmp.Diff(tc.wantSGs, got.GetSecurityGroups()); diff != "" {
				t.Fatalf("security groups (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_parseNetworkAttachmentFlag_WhenInputIsInvalidItShouldError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{name: "When value is empty it should error", input: "   "},
		{name: "When key is unknown it should error", input: "subnet=a,foo=bar"},
		{name: "When subnet= form omits subnet it should error", input: "security-groups=g1"},
		{name: "When fragment has no equals it should error", input: "subnet=sub,noglue"},
		{name: "When value after equals is empty it should error", input: "subnet="},
		{name: "When subnet is duplicated it should error", input: "subnet=a,subnet=b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseNetworkAttachmentFlag(tc.input)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func Test_applyNetworkingFlags_WhenLegacyAndAttachmentsAreCombinedItShouldError(t *testing.T) {
	t.Parallel()
	c := &runnerContext{}
	c.args.subnet = "sub-1"
	c.args.networkAttachments = []string{"other"}
	var b publicv1.ComputeInstanceSpec_builder
	if err := c.applyNetworkingFlags(&b); err == nil {
		t.Fatal("expected error when mixing legacy subnet with network-attachment")
	}
}

func Test_applyNetworkingFlags_WhenLegacySecurityGroupsAndAttachmentsAreCombinedItShouldError(t *testing.T) {
	t.Parallel()
	c := &runnerContext{}
	c.args.securityGroups = []string{"sg-1"}
	c.args.networkAttachments = []string{"sub-x"}
	var b publicv1.ComputeInstanceSpec_builder
	if err := c.applyNetworkingFlags(&b); err == nil {
		t.Fatal("expected error when mixing security-group with network-attachment")
	}
}

func Test_buildSpec_WhenNetworkingFlagsAreSetItShouldPopulateSpec(t *testing.T) {
	t.Parallel()
	c := &runnerContext{}
	c.args.subnet = "legacy-sub"
	c.args.securityGroups = []string{"sg-a", "sg-b"}
	spec, err := c.buildSpec("tmpl", nil)
	if err != nil {
		t.Fatalf("buildSpec: %v", err)
	}
	want := publicv1.ComputeInstanceSpec_builder{
		Template:       "tmpl",
		Subnet:         proto.String("legacy-sub"),
		SecurityGroups: []string{"sg-a", "sg-b"},
	}.Build()
	if diff := cmp.Diff(want, spec, protocmp.Transform()); diff != "" {
		t.Fatalf("spec (-want +got):\n%s", diff)
	}
}

func Test_buildSpec_WhenNetworkAttachmentsAreSetItShouldPopulateAttachments(t *testing.T) {
	t.Parallel()
	c := &runnerContext{}
	c.args.networkAttachments = []string{"n1", "subnet=n2,security-groups=g1"}
	spec, err := c.buildSpec("tmpl", nil)
	if err != nil {
		t.Fatalf("buildSpec: %v", err)
	}
	want := publicv1.ComputeInstanceSpec_builder{
		Template: "tmpl",
		NetworkAttachments: []*publicv1.NetworkAttachment{
			publicv1.NetworkAttachment_builder{Subnet: "n1"}.Build(),
			publicv1.NetworkAttachment_builder{Subnet: "n2", SecurityGroups: []string{"g1"}}.Build(),
		},
	}.Build()
	if diff := cmp.Diff(want, spec, protocmp.Transform()); diff != "" {
		t.Fatalf("spec (-want +got):\n%s", diff)
	}
}
