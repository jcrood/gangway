// Copyright © 2017 Heptio
// Copyright © 2017 Craig Tracey
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

package config

import (
	"os"
	"testing"
)

func TestMultiClusterConfigNotFound(t *testing.T) {
	_, err := NewMultiClusterConfig("nonexistentfile")
	if err == nil {
		t.Errorf("Expected config file parsing to fail for non-existent config file")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Let's assume we're testing the first cluster in the multi-cluster config
	os.Setenv("CLUSTER0_GANGWAY_PROVIDER_URL", "https://foo.bar/authorize")
	os.Setenv("CLUSTER0_GANGWAY_APISERVER_URL", "https://k8s-api.foo.baz")
	os.Setenv("CLUSTER0_GANGWAY_CLIENT_ID", "foo")
	os.Setenv("CLUSTER0_GANGWAY_CLIENT_SECRET", "bar")
	os.Setenv("CLUSTER0_GANGWAY_REDIRECT_URL", "https://foo.baz/callback")
	os.Setenv("CLUSTER0_GANGWAY_SESSION_SECURITY_KEY", "testing")
	os.Setenv("CLUSTER0_GANGWAY_AUDIENCE", "foo")
	os.Setenv("CLUSTER0_GANGWAY_SCOPES", "groups,sub")
	os.Setenv("CLUSTER0_GANGWAY_SHOW_CLAIMS", "false")
	os.Setenv("CLUSTER0_GANGWAY_SESSION_SALT", "randombanana")

	cfg, err := NewMultiClusterConfig("")
	if err != nil {
		t.Errorf("Failed to test config overrides with error: %s", err)
	}
	if cfg == nil || len(cfg.Clusters) == 0 {
		t.Fatalf("No config present")
	}

	clusterConfig := cfg.Clusters[0]
	if clusterConfig.Audience != "foo" {
		t.Errorf("Failed to set audience via environment variable. Expected %s but got %s", "foo", clusterConfig.Audience)
	}

	if clusterConfig.Scopes[0] != "groups" || clusterConfig.Scopes[1] != "sub" {
		t.Errorf("Failed to set scopes via environment variable. Expected %s but got %s", "[groups, sub]", clusterConfig.Scopes)
	}

	if clusterConfig.ShowClaims != false {
		t.Errorf("Failed to disable showing of claims. Expected %t but got %t", false, clusterConfig.ShowClaims)
	}
	if clusterConfig.SessionSalt != "randombanana" {
		t.Errorf("Failed to override session salt. Expected %s but got %s", "randombanana", clusterConfig.SessionSalt)
	}
}

func TestSessionSaltLength(t *testing.T) {
	os.Setenv("CLUSTER0_GANGWAY_PROVIDER_URL", "https://foo.bar")
	os.Setenv("CLUSTER0_GANGWAY_APISERVER_URL", "https://k8s-api.foo.baz")
	os.Setenv("CLUSTER0_GANGWAY_CLIENT_ID", "foo")
	os.Setenv("CLUSTER0_GANGWAY_CLIENT_SECRET", "bar")
	os.Setenv("CLUSTER0_GANGWAY_REDIRECT_URL", "https://foo.baz/callback")
	os.Setenv("CLUSTER0_GANGWAY_SESSION_SECURITY_KEY", "testing")
	os.Setenv("CLUSTER0_GANGWAY_SESSION_SALT", "2short")

	_, err := NewMultiClusterConfig("")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	if err.Error() != "invalid config: salt needs to be min. 8 characters" {
		t.Errorf("Wrong error. Expected %v but got %v", "salt needs to be min. 8 characters", err)
	}
}

func TestGetRootPathPrefix(t *testing.T) {
	tests := map[string]struct {
		path string
		want string
	}{
		"not specified": {
			path: "",
			want: "/",
		},
		"specified": {
			path: "/gangway",
			want: "/gangway",
		},
		"specified default": {
			path: "/",
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &MultiClusterConfig{
				HTTPPath: tc.path,
			}

			got := cfg.GetRootPathPrefix()
			if got != tc.want {
				t.Fatalf("GetRootPathPrefix(): want: %v, got: %v", tc.want, got)
			}
		})
	}
}
