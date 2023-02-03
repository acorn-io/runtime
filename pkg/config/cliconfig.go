package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/acorn-io/acorn/pkg/system"
	"github.com/adrg/xdg"
	"github.com/docker/cli/cli/config/types"
	"sigs.k8s.io/yaml"
)

type AuthConfig types.AuthConfig

func (a AuthConfig) MarshalJSON() ([]byte, error) {
	cp := a
	if cp.Username != "" || cp.Password != "" {
		cp.Auth = base64.StdEncoding.EncodeToString([]byte(cp.Username + ":" + cp.Password))
		cp.Username = ""
		cp.Password = ""
	}
	cp.ServerAddress = ""
	return json.Marshal((types.AuthConfig)(cp))
}

func (a *AuthConfig) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*types.AuthConfig)(a)); err != nil {
		return err
	}
	if a.Auth != "" {
		data, err := base64.StdEncoding.DecodeString(a.Auth)
		if err != nil {
			return err
		}
		a.Username, a.Password, _ = strings.Cut(string(data), ":")
		a.Auth = ""
	}
	return nil
}

//go:generate $GOPATH/bin/mockgen --build_flags=--mod=mod -destination=../mocks/mock_cliconfig.go -package=mocks github.com/acorn-io/acorn/pkg/config CLIConfig
type CLIConfig struct {
	Auths             map[string]AuthConfig `json:"auths,omitempty"`
	CredentialsStore  string                `json:"credsStore,omitempty"`
	CredentialHelpers map[string]string     `json:"credHelpers,omitempty"`
	HubServers        []string              `json:"hubServers,omitempty"`
	Kubeconfigs       map[string]string     `json:"-"`
	ProjectAliases    map[string]string     `json:"projectAliases,omitempty"`
	CurrentProject    string                `json:"currentProject,omitempty"`

	// TestProjectURLs is used for testing to return EndpointURLs for remote projects
	TestProjectURLs map[string]string `json:"-"`

	filename  string
	auths     map[string]types.AuthConfig
	authsLock *sync.Mutex
}

func (c *CLIConfig) Sanitize() *CLIConfig {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Auths = map[string]AuthConfig{}
	for k := range c.Auths {
		cp.Auths[k] = AuthConfig{
			Auth: "<redacted>",
		}
	}
	return &cp
}

func (c *CLIConfig) Save() error {
	if c.authsLock != nil {
		c.authsLock.Lock()
		defer c.authsLock.Unlock()
	}

	if c.auths != nil {
		c.Auths = map[string]AuthConfig{}
		for k, v := range c.auths {
			c.Auths[k] = (AuthConfig)(v)
		}
		c.auths = nil
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.filename, data, 0655)
}

func (c *CLIConfig) GetAuthConfigs() map[string]types.AuthConfig {
	if c.authsLock != nil {
		c.authsLock.Lock()
		defer c.authsLock.Unlock()
	}

	if c.auths == nil {
		c.auths = map[string]types.AuthConfig{}
		for k, v := range c.Auths {
			authConfig := (types.AuthConfig)(v)
			c.auths[k] = authConfig
		}
	}
	return c.auths
}

func (c *CLIConfig) GetFilename() string {
	return c.filename
}

func ReadCLIConfig() (*CLIConfig, error) {
	filename, err := CLIConfigFile()
	if err != nil {
		return nil, err
	}
	kubeconfigDir, err := KubeconfigDir()
	if err != nil {
		return nil, err
	}
	data, err := readFile(filename)
	if err != nil {
		return nil, err
	}
	result := &CLIConfig{
		authsLock: &sync.Mutex{},
	}
	if err := yaml.Unmarshal(data, result); err != nil {
		return nil, err
	}

	result.filename = filename

	if len(result.HubServers) == 0 {
		result.HubServers = []string{system.DefaultHubAddress}
	}

	result.Kubeconfigs = map[string]string{}

	entries, err := os.ReadDir(kubeconfigDir)
	if os.IsNotExist(err) {
	} else if err != nil {
		return nil, err
	} else {
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				name = name[:len(name)-len(filepath.Ext(name))]
				result.Kubeconfigs[name] = filepath.Join(kubeconfigDir, entry.Name())
			}
		}
	}

	return result, nil
}

func KubeconfigDir() (string, error) {
	cfg, err := CLIConfigFile()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "kubeconfigs"), nil
}

func CLIConfigFile() (string, error) {
	var (
		location = os.Getenv("ACORN_CONFIG_FILE")
		err      error
	)

	if location == "" {
		location, err = xdg.ConfigFile("acorn/config.yaml")
		if err != nil {
			return "", fmt.Errorf("failed to read user config from standard location: %w", err)
		}
	}

	return location, nil
}

func readFile(location string) ([]byte, error) {
	data, err := os.ReadFile(location)
	if os.IsNotExist(err) {
		return []byte("{}"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read user config %s: %w", location, err)
	}

	return data, nil
}

func RemoveServer(cfg *CLIConfig, serverAddress string) error {
	var modified bool
	if strings.HasPrefix(cfg.CurrentProject, serverAddress) {
		cfg.CurrentProject = ""
		modified = true
	}

	var newHubServers []string
	for _, server := range cfg.HubServers {
		if server != serverAddress {
			newHubServers = append(newHubServers, server)
		}
	}

	if len(newHubServers) != len(cfg.HubServers) {
		cfg.HubServers = newHubServers
		modified = true
	}

	if modified {
		return cfg.Save()
	}

	return nil
}
