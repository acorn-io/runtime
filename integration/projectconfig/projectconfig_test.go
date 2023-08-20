package projectconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
)

func TestSlashBreaksList(t *testing.T) {
	p, _, err := project.List(context.Background(), false, project.Options{
		Project: "acorn.io/fake/fake",
	})
	assert.Nil(t, err)
	assert.True(t, len(p) > 0)
}

func TestCLIConfig(t *testing.T) {
	d, err := os.MkdirTemp("", "acorn-test-home")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(d)
	})

	xdg.Home = d
	testAPIKubeconfig := testRestConfig(t, "testhost", "")
	testAPIKubeconfig2 := testRestConfig(t, "testhost2", "")
	//testAPIKubeconfigEnv := testRestConfig(t, "testenv", "")

	tests := []struct {
		name               string
		opt                project.Options
		cliConfig          config.CLIConfig
		kubeconfigEnv      string
		homeEnv            string
		noEnv              bool
		wantProject        string
		wantNamespace      string
		wantRestConfigHost string
		wantToken          string
		assert             func(*testing.T, client.Client)
		wantError          bool
		wantErr            error
	}{
		{
			name: "User set multiple files in kubeconfig env",
			kubeconfigEnv: fmt.Sprintf("%s%s%s", testAPIKubeconfig,
				[]byte{filepath.ListSeparator},
				testAPIKubeconfig2),
			opt: project.Options{
				ContextEnv: "testhost2",
			},
			wantNamespace:      "acorn",
			wantRestConfigHost: "testhost2",
		},
		{
			name: "User passes --kubeconfig",
			opt: project.Options{
				Kubeconfig: testAPIKubeconfig,
			},
			wantNamespace:      "acorn",
			wantRestConfigHost: "testhost",
		},
		{
			name: "User passes --kubeconfig with namespace set in it, we ignore it",
			opt: project.Options{
				Kubeconfig: testRestConfig(t, "testhost", "testnamespace"),
			},
			wantRestConfigHost: "testhost",
			wantNamespace:      "acorn",
		},
		{
			name: "User passes --kubeconfig with --project set",
			opt: project.Options{
				Kubeconfig: testRestConfig(t, "testhost", "testnamespace"),
				Project:    "projectnamespace",
			},
			wantRestConfigHost: "testhost",
			wantNamespace:      "projectnamespace",
		},
		{
			name: "User passes --kubeconfig with current project set",
			opt: project.Options{
				Kubeconfig: testRestConfig(t, "testhost", "testnamespace"),
				Project:    "projectnamespace",
			},
			cliConfig: config.CLIConfig{
				CurrentProject: "foo/bar",
			},
			wantRestConfigHost: "testhost",
			wantNamespace:      "projectnamespace",
		},
		{
			name: "Current project is external",
			opt:  project.Options{},
			cliConfig: config.CLIConfig{
				CurrentProject: "example.com/foo/bar",
				ProjectURLs: map[string]string{
					"example.com/foo": "https://foo.example.com",
				},
				Auths: map[string]config.AuthConfig{
					"example.com": {
						Password: "pass",
					},
				},
			},
			wantRestConfigHost: "https://foo.example.com",
			wantNamespace:      "bar",
			wantToken:          "pass",
		},
		{
			name: "Project arg overrides current project",
			opt: project.Options{
				Project:    "projectnamespace",
				Kubeconfig: testAPIKubeconfig,
			},
			cliConfig: config.CLIConfig{
				CurrentProject: "example.com/foo/bar",
			},
			wantNamespace: "projectnamespace",
		},
		{
			name:          "No config",
			wantError:     true,
			noEnv:         true,
			kubeconfigEnv: "garbage",
			homeEnv:       "garbage",
		},
		{
			name: "No config, but user requested project",
			opt: project.Options{
				Project: "something",
			},
			kubeconfigEnv: testAPIKubeconfig,
			wantNamespace: "something",
		},
		{
			name: "User set manager reference",
			opt: project.Options{
				Project: "example.com/foo/bar",
			},
			cliConfig: config.CLIConfig{
				Auths: map[string]config.AuthConfig{
					"example.com": {
						Password: "pass",
					},
				},
				ProjectURLs: map[string]string{
					"example.com/foo": "https://endpoint.example.com",
				},
			},
			wantRestConfigHost: "https://endpoint.example.com",
			wantProject:        "example.com/foo/bar",
			wantNamespace:      "bar",
			wantToken:          "pass",
		},
		{
			name:          "Use alias",
			kubeconfigEnv: testAPIKubeconfig,
			cliConfig: config.CLIConfig{
				ProjectAliases: map[string]string{
					"foo": "kubeconfig/ns2,example.com/acct/prj1,defns",
				},
				Auths: map[string]config.AuthConfig{
					"example.com": {
						Password: "pass",
					},
				},
				ProjectURLs: map[string]string{
					"example.com/acct": "https://endpoint.example.com",
				},
			},
			opt: project.Options{
				Project: "foo",
			},
			assert: func(t *testing.T, c client.Client) {
				t.Helper()
				clients, err := c.(*client.MultiClient).Factory.List(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				assert.Len(t, clients, 3)
			},
			wantProject:   "kubeconfig/ns2,example.com/acct/prj1,kubeconfig/defns",
			wantNamespace: "ns2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldEnv := os.Environ()
			oldHome := os.Getenv("HOME")
			oldKubeconfig := os.Getenv("KUBECONFIG")
			oldRecommendHomeFile := clientcmd.RecommendedHomeFile
			oldAcornConfig := os.Getenv("ACORN_CONFIG_FILE")
			if test.noEnv {
				for _, env := range oldEnv {
					k, _, _ := strings.Cut(env, "=")
					os.Setenv(k, "")
				}
			}
			tmp, err := os.CreateTemp("", "acorn-config")
			if err != nil {
				t.Fatal(err)
			}
			if err := json.NewEncoder(tmp).Encode(test.cliConfig); err != nil {
				t.Fatal(err)
			}
			if err := tmp.Close(); err != nil {
				t.Fatal(err)
			}
			os.Setenv("ACORN_CONFIG_FILE", tmp.Name())
			defer func() {
				_ = os.Remove(tmp.Name())
			}()
			if test.kubeconfigEnv != "" {
				os.Setenv("KUBECONFIG", test.kubeconfigEnv)
			}
			if test.homeEnv != "" {
				os.Setenv("HOME", test.homeEnv)
				clientcmd.RecommendedHomeFile = filepath.Join(test.homeEnv, ".kube", "config")
			}
			c, err := testCLIConfig(t, test.opt)
			assert.Equal(t, test.wantError, err != nil, "should have error")
			for _, env := range oldEnv {
				k, v, _ := strings.Cut(env, "=")
				os.Setenv(k, v)
			}
			os.Setenv("KUBECONFIG", oldKubeconfig)
			os.Setenv("HOME", oldHome)
			os.Setenv("ACORN_CONFIG_FILE", oldAcornConfig)
			clientcmd.RecommendedHomeFile = oldRecommendHomeFile
			if test.wantErr != nil {
				assert.Equal(t, test.wantErr, err)
			}
			if err != nil {
				if !test.wantError {
					t.Fatal(err)
				}
				return
			}
			if test.wantNamespace != "" {
				assert.Equal(t, test.wantNamespace, getDefaultClient(t, c).GetNamespace())
			}
			if test.wantProject != "" {
				assert.Equal(t, test.wantProject, c.GetProject())
			}
			if test.wantRestConfigHost != "" {
				assert.Equal(t, test.wantRestConfigHost, getDefaultClient(t, c).RESTConfig.Host)
			}
			if test.wantToken != "" {
				assert.Equal(t, test.wantToken, getDefaultClient(t, c).RESTConfig.BearerToken)
			}
			if test.assert != nil {
				test.assert(t, c)
			}
		})
	}
}

func getDefaultClient(t *testing.T, c client.Client) *client.DefaultClient {
	t.Helper()

	if mc, ok := c.(*client.MultiClient); ok {
		var err error
		c, err = mc.Factory.ForProject(context.Background(), mc.Factory.DefaultProject())
		if err != nil {
			t.Fatal(err)
		}
	}
	if d, ok := c.(*client.DeferredClient); ok {
		if d.Client == nil {
			newClient, err := d.New()
			if err != nil {
				t.Fatal(err)
			}
			c = newClient
		} else {
			c = d.Client
		}
	}
	return c.(*client.DefaultClient)
}

func testRestConfig(t *testing.T, host, namespace string) string {
	t.Helper()

	//cfg := helper.StartAPI(t)
	tempDir, err := os.MkdirTemp("", "acorn-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	filename := filepath.Join(tempDir, "kubeconfig.yaml")
	err = os.WriteFile(filename, []byte(strings.ReplaceAll(fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: "%s"
  name: testingDefault
contexts:
- context:
    cluster: testingDefault
    user: testingDefault
    namespace: "%s"
  name: testingDefault
current-context: testingDefault
kind: Config
users:
- name: testingDefault
  user:
    token: ""
`, host, namespace), "testingDefault", "\""+host+"\"")), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return filename
}

func testCLIConfig(t *testing.T, opt project.Options) (client.Client, error) {
	t.Helper()

	return project.Client(context.Background(), opt)
}
