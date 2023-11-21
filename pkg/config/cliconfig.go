package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/acorn-io/runtime/pkg/system"
	"github.com/adrg/xdg"
	"github.com/docker/cli/cli/config/types"
	"sigs.k8s.io/yaml"
)

const (
	LocalServer    = "kubeconfig"
	LocalServerEnv = "local " + LocalServer
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

type CLIConfig struct {
	Auths              map[string]AuthConfig `json:"auths,omitempty"`
	CredentialsStore   string                `json:"credsStore,omitempty"`
	CredentialHelpers  map[string]string     `json:"credHelpers,omitempty"`
	DefaultAcornServer string                `json:"defaultAcornServer,omitempty"`
	AcornServers       []string              `json:"acornServers,omitempty"`
	ProjectAliases     map[string]string     `json:"projectAliases,omitempty"`
	DefaultContext     string                `json:"defaultContext,omitempty"`
	CurrentProject     string                `json:"currentProject,omitempty"`
	LastProject        string                `json:"lastProject,omitempty"`
	AcornConfigFile    string                `json:"acornConfig,omitempty"`

	// ProjectURLs is used for testing to return EndpointURLs for remote projects
	ProjectURLs map[string]string `json:"projectURLs,omitempty"`

	auths     map[string]types.AuthConfig
	authsLock *sync.Mutex
}

func (c *CLIConfig) GetDefaultAcornServer() string {
	if c == nil || c.DefaultAcornServer == "" {
		return system.DefaultManagerAddress
	}
	return c.DefaultAcornServer
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
	return os.WriteFile(c.AcornConfigFile, data, 0655)
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
	return c.AcornConfigFile
}

func ReadCLIConfig(acornConfigFile string, kubeconfigOnly bool) (*CLIConfig, error) {
	if acornConfigFile == "" {
		// If acornConfigFile isn't provided, check the environment variable
		if acornConfigFile = os.Getenv("ACORN_CONFIG_FILE"); acornConfigFile == "" {
			// If an environment variable isn't provided, check the default location
			var err error
			if acornConfigFile, err = xdg.ConfigFile("acorn/config.yaml"); err != nil {
				return nil, fmt.Errorf("failed to read user config from standard location: %w", err)
			}
		}
	}

	data, err := readFile(acornConfigFile)
	if err != nil {
		return nil, err
	}
	result := &CLIConfig{
		authsLock:       &sync.Mutex{},
		AcornConfigFile: acornConfigFile,
	}
	if err := yaml.Unmarshal(data, result); err != nil {
		return nil, err
	}

	if len(result.AcornServers) == 0 {
		result.AcornServers = []string{result.GetDefaultAcornServer()}
	}

	if kubeconfigOnly {
		result.DefaultContext = ""
		result.CurrentProject = ""
		result.LastProject = ""
	}

	return result, nil
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []byte("{}"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read user config %s: %w", path, err)
	}

	return data, nil
}

func RemoveServer(cfg *CLIConfig, serverAddress string) error {
	var modified bool
	if strings.HasPrefix(cfg.CurrentProject, serverAddress) {
		cfg.CurrentProject = ""
		modified = true
	}

	if strings.HasPrefix(cfg.LastProject, serverAddress) {
		cfg.LastProject = ""
		modified = true
	}

	var newAcornServer []string
	for _, server := range cfg.AcornServers {
		if server != serverAddress {
			newAcornServer = append(newAcornServer, server)
		}
	}

	if len(newAcornServer) != len(cfg.AcornServers) {
		cfg.AcornServers = newAcornServer
		modified = true
	}

	if strings.HasPrefix(cfg.DefaultContext, serverAddress+"/") {
		cfg.DefaultContext = ""
		modified = true
	}

	if modified {
		return cfg.Save()
	}

	return nil
}
