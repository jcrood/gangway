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

func TestConfigNotFound(t *testing.T) {
	_, err := NewConfig("nonexistentfile")
	if err == nil {
		t.Errorf("Expected config file parsing to file for non-existent config file")
	}
}

func TestEnvionmentOverrides(t *testing.T) {
	salt := "randombanana"
	os.Setenv("GANGWAY_PROVIDER_URL", "https://foo.bar/authorize")
	os.Setenv("GANGWAY_APISERVER_URL", "https://k8s-api.foo.baz")
	os.Setenv("GANGWAY_CLIENT_ID", "foo")
	os.Setenv("GANGWAY_CLIENT_SECRET", "bar")
	os.Setenv("GANGWAY_PORT", "1234")
	os.Setenv("GANGWAY_REDIRECT_URL", "https://foo.baz/callback")
	os.Setenv("GANGWAY_CLUSTER_CA_PATH", "") // FIXME: add test fixture
	os.Setenv("GANGWAY_SESSION_SECURITY_KEY", "testing")
	os.Setenv("GANGWAY_AUDIENCE", "foo")
	os.Setenv("GANGWAY_SCOPES", "groups,sub")
	os.Setenv("GANGWAY_SHOW_CLAIMS", "false")
	os.Setenv("GANGWAY_SESSION_SALT", salt)
	cfg, err := NewConfig("")
	if err != nil {
		t.Errorf("Failed to test config overrides with error: %s", err)
	}
	if cfg == nil {
		t.Fatalf("No config present")
	}

	if cfg.Port != 1234 {
		t.Errorf("Failed to override config with environment")
	}

	if cfg.Audience != "foo" {
		t.Errorf("Failed to set audience via environment variable. Expected %s but got %s", "foo", cfg.Audience)
	}

	if cfg.Scopes[0] != "groups" || cfg.Scopes[1] != "sub" {
		t.Errorf("Failed to set scopes via environment variable. Expected %s but got %s", "[groups, sub]", cfg.Scopes)
	}

	if cfg.ShowClaims != false {
		t.Errorf("Failed to disable showing of claims. Expected %t but got %t", false, cfg.ShowClaims)
	}
	if cfg.SessionSalt != salt {
		t.Errorf("Failed to override session salt. Expected %s but got %s", salt, cfg.SessionSalt)
	}
}

func TestSessionSaltLength(t *testing.T) {
	salt := "2short"
	os.Setenv("GANGWAY_PROVIDER_URL", "https://foo.bar")
	os.Setenv("GANGWAY_APISERVER_URL", "https://k8s-api.foo.baz")
	os.Setenv("GANGWAY_CLIENT_ID", "foo")
	os.Setenv("GANGWAY_CLIENT_SECRET", "bar")
	os.Setenv("GANGWAY_REDIRECT_URL", "https://foo.baz/callback")
	os.Setenv("GANGWAY_SESSION_SECURITY_KEY", "testing")
	os.Setenv("GANGWAY_SESSION_SALT", salt)
	_, err := NewConfig("")
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
			cfg := &Config{
				HTTPPath: tc.path,
			}

			got := cfg.GetRootPathPrefix()
			if got != tc.want {
				t.Fatalf("GetRootPathPrefix(): want: %v, got: %v", tc.want, got)
			}
		})
	}
}
