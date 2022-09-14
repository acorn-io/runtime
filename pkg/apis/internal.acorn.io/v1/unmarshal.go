package v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/shlex"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	DefaultSizeQuantity = Quantity("10G")
	MinSizeQuantity     = Quantity("5M")
	DefaultSize         = MustParseResourceQuantity(DefaultSizeQuantity)
	MinSize             = MustParseResourceQuantity(MinSizeQuantity)
)

func (in *NameValue) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		type nameValue NameValue
		return json.Unmarshal(data, (*nameValue)(in))
	}
	s, err := parseString(data)
	if err != nil {
		return err
	}
	*in = ParseNameValues(false, s)[0]
	return nil
}

func (in *NameValues) UnmarshalJSON(data []byte) error {
	if !isObject(data) {
		return json.Unmarshal(data, (*[]NameValue)(in))
	}

	nv := map[string]string{}
	if err := json.Unmarshal(data, &nv); err != nil {
		return err
	}
	for k, v := range nv {
		*in = append(*in, NameValue{Name: k, Value: v})
	}
	sort.Slice(*in, func(i, j int) bool {
		if (*in)[i].Name == (*in)[j].Name {
			return (*in)[i].Value < (*in)[j].Value
		}
		return (*in)[i].Name < (*in)[j].Name
	})
	return nil
}

func (in *Dependencies) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		return json.Unmarshal(data, (*[]Dependency)(in))
	}
	var dep Dependency
	if err := json.Unmarshal(data, &dep); err != nil {
		return err
	}
	*in = append(*in, dep)
	return nil
}

func (in *Quantity) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		var s int
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s < 1000000 {
			*in = (Quantity)(fmt.Sprintf("%dG", s))
		} else {
			*in = (Quantity)(fmt.Sprintf("%d", s))
		}
		return nil
	}
	s, err := parseString(data)
	if err != nil {
		return err
	}
	q, err := ParseQuantity(s)
	if err != nil {
		return err
	}
	*in = q
	return nil
}

func (in *ServiceBinding) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		type serviceBinding ServiceBinding
		return json.Unmarshal(data, (*serviceBinding)(in))
	}

	s, err := parseString(data)
	if err != nil {
		return err
	}
	result, err := ParseLinks([]string{s})
	if err != nil {
		return nil
	}

	*in = result[0]
	return nil
}

func (in *SecretBinding) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		type secretBinding SecretBinding
		return json.Unmarshal(data, (*secretBinding)(in))
	}

	s, err := parseString(data)
	if err != nil {
		return err
	}
	result, err := ParseSecrets([]string{s})
	if err != nil {
		return err
	}
	*in = result[0]
	return nil
}

func (in *VolumeBinding) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		type volumeBinding VolumeBinding
		return json.Unmarshal(data, (*volumeBinding)(in))
	}

	s, err := parseString(data)
	if err != nil {
		return err
	}
	result, err := ParseVolumes([]string{s}, false)
	if err != nil {
		return err
	}
	*in = result[0]
	return nil
}

func impliedSecretsForContainer(app *AppSpec, container Container) {
	for _, env := range container.Environment {
		if _, ok := app.Secrets[env.Secret.Name]; env.Secret.Name != "" && !ok {
			app.Secrets[env.Secret.Name] = Secret{
				Type: "opaque",
			}
		}
	}
	for _, dir := range container.Dirs {
		if _, ok := app.Secrets[dir.Secret.Name]; dir.Secret.Name != "" && !ok {
			app.Secrets[dir.Secret.Name] = Secret{
				Type: "opaque",
			}
		}
	}
	for _, file := range container.Files {
		if _, ok := app.Secrets[file.Secret.Name]; file.Secret.Name != "" && !ok {
			app.Secrets[file.Secret.Name] = Secret{
				Type: "opaque",
			}
		}
	}
}

func impliedVolumesForContainer(app *AppSpec, containerName, sideCarName string, container Container) error {
	for path, mount := range container.Dirs {
		if mount.ContextDir != "" || mount.Secret.Name != "" {
			continue
		}

		if strings.HasPrefix(mount.Volume, "volume://") || strings.HasPrefix(mount.Volume, "ephemeral://") || mount.Volume == "" {
			v, err := parseVolumeDefinition(filepath.Join(containerName, sideCarName, path), mount.Volume)
			if err != nil {
				return err
			}
			mount.Volume = v.Volume
			container.Dirs[path] = mount
			if existing, ok := app.Volumes[mount.Volume]; ok {
				existingSize, err := resource.ParseQuantity((string)(existing.Size))
				if err != nil {
					// ignore error
					continue
				}
				vSize, err := resource.ParseQuantity((string)(v.Size))
				if err != nil {
					// ignore error
					continue
				}
				if existingSize.Cmp(vSize) < 0 {
					existing.Size = v.Size
				}
				for _, accessMode := range v.AccessModes {
					found := false
					for _, existingAccessMode := range existing.AccessModes {
						if existingAccessMode == accessMode {
							found = true
							break
						}
					}
					if !found {
						existing.AccessModes = append(existing.AccessModes, accessMode)
					}
				}
				sort.Slice(existing.AccessModes, func(i, j int) bool {
					return existing.AccessModes[i] < existing.AccessModes[j]
				})
				app.Volumes[mount.Volume] = existing
			} else {
				app.Volumes[mount.Volume] = VolumeRequest{
					Size:        v.Size,
					AccessModes: v.AccessModes,
					Class:       v.Class,
				}
			}
		} else if _, ok := app.Volumes[mount.Volume]; !ok {
			app.Volumes[mount.Volume] = VolumeRequest{
				Size:        DefaultSizeQuantity,
				AccessModes: []AccessMode{AccessModeReadWriteOnce},
			}
		}
	}
	return nil
}

func checkForDuplicateNames(in *AppSpec) error {
	names := map[string]string{}
	for name, c := range in.Containers {
		for _, port := range c.Ports {
			if port.ServiceName != "" && port.ServiceName != name {
				if err := addName(names, port.ServiceName, "port"); err != nil {
					return err
				}
			}
		}
		if err := addName(names, name, "container"); err != nil {
			return err
		}
		for _, sidecar := range c.Sidecars {
			for _, port := range sidecar.Ports {
				if port.ServiceName != "" && port.ServiceName != name {
					if err := addName(names, port.ServiceName, "port"); err != nil {
						return err
					}
				}
			}
		}
	}
	for name := range in.Jobs {
		if err := addName(names, name, "job"); err != nil {
			return err
		}
	}
	for name, a := range in.Acorns {
		for _, port := range a.Ports {
			if port.ServiceName != "" && port.ServiceName != name {
				if err := addName(names, port.ServiceName, "port"); err != nil {
					return err
				}
			}
		}
		if err := addName(names, name, "acorn"); err != nil {
			return err
		}
	}

	return nil
}

func addImpliedResources(in *AppSpec) error {
	if in.Volumes == nil {
		in.Volumes = map[string]VolumeRequest{}
	}
	if in.Secrets == nil {
		in.Secrets = map[string]Secret{}
	}

	for _, a := range in.Acorns {
		for _, volumeBinding := range a.Volumes {
			if _, ok := in.Volumes[volumeBinding.Volume]; !ok {
				in.Volumes[volumeBinding.Volume] = VolumeRequest{
					Size:        volumeBinding.Size,
					AccessModes: volumeBinding.AccessModes,
				}
			}
		}
		for _, secretBinding := range a.Secrets {
			if _, ok := in.Secrets[secretBinding.Secret]; !ok {
				in.Secrets[secretBinding.Secret] = Secret{
					Type: "opaque",
				}
			}
		}
	}

	for containerName, c := range in.Containers {
		impliedSecretsForContainer(in, c)
		if err := impliedVolumesForContainer(in, containerName, "", c); err != nil {
			return err
		}
		for sidecarName, s := range c.Sidecars {
			impliedSecretsForContainer(in, s)
			if err := impliedVolumesForContainer(in, containerName, sidecarName, s); err != nil {
				return err
			}
		}
	}

	for containerName, j := range in.Jobs {
		impliedSecretsForContainer(in, j)
		if err := impliedVolumesForContainer(in, containerName, "", j); err != nil {
			return err
		}
		for sidecarName, s := range j.Sidecars {
			impliedSecretsForContainer(in, s)
			if err := impliedVolumesForContainer(in, containerName, sidecarName, s); err != nil {
				return err
			}
		}
	}

	return nil
}

func (in *AppSpec) UnmarshalJSON(data []byte) error {
	type appSpec AppSpec
	if err := json.Unmarshal(data, (*appSpec)(in)); err != nil {
		return err
	}

	if err := addImpliedResources(in); err != nil {
		return err
	}

	return checkForDuplicateNames(in)
}

func addName(data map[string]string, key, value string) error {
	existing := data[key]
	if existing != "" && existing != value {
		return fmt.Errorf("duplicate name [%s] used by [%s] and [%s]", key, existing, value)
	}
	data[key] = value
	return nil
}

func (in *ContainerImageBuilderSpec) UnmarshalJSON(data []byte) error {
	var container Container
	if err := json.Unmarshal(data, &container); err != nil {
		return err
	}

	in.Image = container.Image
	in.Build = container.Build
	if len(container.Sidecars) > 0 {
		in.Sidecars = map[string]ContainerImageBuilderSpec{}
		for name, sidecar := range container.Sidecars {
			in.Sidecars[name] = ContainerImageBuilderSpec{
				Image: sidecar.Image,
				Build: sidecar.Build,
			}
		}
	}

	return nil
}

func (in *AccessModes) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		return json.Unmarshal(data, (*[]AccessMode)(in))
	}
	var mode AccessMode
	if err := json.Unmarshal(data, &mode); err != nil {
		return err
	}
	*in = append(*in, mode)
	return nil
}

type acornAliases struct {
	Env NameValues `json:"env,omitempty"`
}

func (a acornAliases) SetAcorn(dst Acorn) Acorn {
	if len(a.Env) > 0 {
		dst.Environment = append(dst.Environment, a.Env...)
	}
	return dst
}

type containerAliases struct {
	Cmd                 CommandSlice           `json:"cmd,omitempty"`
	Env                 EnvVars                `json:"env,omitempty"`
	WorkDir             string                 `json:"workDir,omitempty"`
	TTY                 bool                   `json:"tty,omitempty"`
	Stdin               bool                   `json:"stdin,omitempty"`
	Probe               Probes                 `json:"probe,omitempty"`
	Directories         map[string]VolumeMount `json:"directories,omitempty"`
	DependsOn           Dependencies           `json:"dependsOn,omitempty"`
	DependsOnUnderscore Dependencies           `json:"depends_on,omitempty"`
}

func (c containerAliases) SetContainer(dst Container) Container {
	if len(c.Cmd) > 0 {
		dst.Command = c.Cmd
	}
	if len(c.Env) > 0 {
		dst.Environment = append(dst.Environment, c.Env...)
	}
	if c.WorkDir != "" {
		dst.WorkingDir = c.WorkDir
	}
	if c.TTY {
		dst.Interactive = true
	}
	if c.Stdin {
		dst.Interactive = true
	}
	if len(c.Probe) > 0 {
		dst.Probes = c.Probe
	}
	if len(c.Directories) > 0 {
		dst.Dirs = c.Directories
	}
	if len(c.DependsOn) > 0 {
		dst.Dependencies = c.DependsOn
	}
	if len(c.DependsOnUnderscore) > 0 {
		dst.Dependencies = c.DependsOnUnderscore
	}
	return dst
}

func adjustBuildForContextDirs(c Container) *Build {
	dirs := map[string]string{}
	build := c.Build
	for path, dir := range c.Dirs {
		if dir.ContextDir != "" {
			dirs[path] = dir.ContextDir
		}
	}

	if len(dirs) == 0 {
		return build
	}

	if build == nil {
		build = &Build{
			Context:    ".",
			Dockerfile: "Dockerfile",
		}
	}
	build.BaseImage = c.Image
	build.ContextDirs = dirs
	return build
}

func (in *Acorn) UnmarshalJSON(data []byte) error {
	var a Acorn
	type acorn Acorn
	if err := json.Unmarshal(data, (*acorn)(&a)); err != nil {
		return err
	}

	var alias acornAliases
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	a = alias.SetAcorn(a)
	*in = a
	return nil
}

func (in *Container) UnmarshalJSON(data []byte) error {
	var c Container
	type container Container
	if err := json.Unmarshal(data, (*container)(&c)); err != nil {
		return err
	}

	var alias containerAliases
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	c = alias.SetContainer(c)

	c.Build = adjustBuildForContextDirs(c)
	for name, sidecar := range c.Sidecars {
		sidecar.Build = adjustBuildForContextDirs(sidecar)
		c.Sidecars[name] = sidecar
	}

	*in = c
	return nil
}

func (in *PolicyRule) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		return json.Unmarshal(data, (*rbacv1.PolicyRule)(in))
	}

	s, err := parseString(data)
	if err != nil {
		return err
	}
	read := strings.HasPrefix(s, "read ")
	if read {
		s = strings.TrimPrefix(s, "read ")
	}

	resource, apiGroup, _ := strings.Cut(s, ".")
	in.Resources = []string{resource}
	in.APIGroups = []string{apiGroup}
	in.Verbs = []string{"*"}
	if read {
		in.Verbs = []string{"get", "list", "watch"}
	}

	return nil
}

func (in *Permissions) UnmarshalJSON(data []byte) error {
	if !isArray(data) {
		type permissions Permissions
		return json.Unmarshal(data, (*permissions)(in))
	}

	var rules []PolicyRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}
	in.Rules = rules
	return nil
}

func (in *Dependency) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		in.TargetName = s
		return nil
	}

	type dependency Dependency
	return json.Unmarshal(data, (*dependency)(in))
}

func (in *PortDef) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		ports, err := ParsePorts([]string{s})
		if err != nil {
			return err
		}
		*in = ports[0]
		return nil
	} else if !isObject(data) {
		var num int32
		if err := json.Unmarshal(data, &num); err != nil {
			return err
		}
		in.TargetPort = num
		in.Port = num
		return nil
	}

	type portDef PortDef
	return json.Unmarshal(data, (*portDef)(in))
}

func (in *Ports) UnmarshalJSON(data []byte) error {
	if isObject(data) {
		ports := map[string]Ports{}
		if err := json.Unmarshal(data, &ports); err != nil {
			return err
		}
		for _, port := range ports["expose"] {
			port.Expose = true
			*in = append(*in, port)
		}
		*in = append(*in, ports["internal"]...)
		for _, port := range ports["publish"] {
			port.Publish = true
			*in = append(*in, port)
		}
		return nil
	} else if isArray(data) {
		return json.Unmarshal(data, (*[]PortDef)(in))
	} else if isString(data) {
		var p PortDef
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		*in = append(*in, p)
		return nil
	}

	// number
	var num int32
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*in = append(*in, PortDef{
		TargetPort: num,
		Port:       num,
	})
	return nil
}

func (in *VolumeMount) UnmarshalJSON(data []byte) error {
	if !isString(data) {
		type volumeMount VolumeMount
		return json.Unmarshal(data, (*volumeMount)(in))
	}

	s, err := parseString(data)
	if err != nil {
		return err
	}

	sec, ok, err := parseSecretReference(s)
	if err != nil {
		return err
	}

	if ok {
		in.Secret.Name = sec.SecretReference.Name
		in.Secret.OnChange = sec.SecretReference.OnChange
	} else if strings.HasPrefix(s, "./") {
		in.ContextDir = s
	} else {
		in.Volume, in.SubPath, err = parseVolumeReference(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (in *Probe) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}

		in.Type = ReadinessProbeType
		in.TimeoutSeconds = 1
		in.PeriodSeconds = 10
		in.SuccessThreshold = 1
		in.FailureThreshold = 3

		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			in.HTTP = &HTTPProbe{
				URL: s,
			}
		} else if strings.HasPrefix(s, "tcp://") {
			in.TCP = &TCPProbe{
				URL: s,
			}
		} else {
			cmd, err := shlex.Split(s)
			if err != nil {
				return fmt.Errorf("parsing command slice %s: %w", s, err)
			}
			in.Exec = &ExecProbe{
				Command: cmd,
			}
		}
		return nil
	}

	type probe Probe
	err := json.Unmarshal(data, (*probe)(in))
	if err != nil {
		return err
	}
	if in.Type == "ready" {
		in.Type = ReadinessProbeType
	}
	return nil
}

func (in *Probes) UnmarshalJSON(data []byte) error {
	// ensure not nil if set
	*in = Probes{}

	if isString(data) {
		p := Probe{}
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		*in = append(*in, p)
		return nil
	} else if isObject(data) {
		d := map[string]Probe{}
		if err := json.Unmarshal(data, &d); err != nil {
			return err
		}
		for k, v := range d {
			v.Type = ProbeType(k)
			if v.Type == "ready" {
				v.Type = ReadinessProbeType
			}
			*in = append(*in, v)
		}
		sort.Slice(*in, func(i, j int) bool {
			return (*in)[i].Type < (*in)[j].Type
		})
		return nil
	}

	type probes Probes
	return json.Unmarshal(data, (*probes)(in))
}

type envVal struct {
	Name   string          `json:"name,omitempty"`
	Value  string          `json:"value,omitempty"`
	Secret SecretReference `json:"secret,omitempty"`
}

func (in *envVal) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		envVar, err := parseEnvVar("", s)
		if err != nil {
			return fmt.Errorf("parsing env var value %s: %w", s, err)
		}
		*in = (envVal)(envVar)
		return nil
	}

	type envValue envVal
	return json.Unmarshal(data, (*envValue)(in))
}

func (in *EnvVar) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		k, v, _ := strings.Cut(s, "=")
		envVar, err := parseEnvVar(k, v)
		if err != nil {
			return fmt.Errorf("parsing env var %s=%s: %w", k, v, err)
		}
		*in = envVar
		return nil
	}

	type envVar EnvVar
	return json.Unmarshal(data, (*envVar)(in))
}

func (in *EnvVars) UnmarshalJSON(data []byte) error {
	if isObject(data) {
		values := map[string]envVal{}
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for k, v := range values {
			sec, ok, err := parseSecretReference(k)
			if err != nil {
				return err
			}
			if ok {
				v.Secret = sec.SecretReference
			} else {
				v.Name = k
			}
			*in = append(*in, (EnvVar)(v))
		}
	} else if err := json.Unmarshal(data, (*[]EnvVar)(in)); err != nil {
		return err
	}

	sort.Slice(*in, func(i, j int) bool {
		return (*in)[i].Name < (*in)[j].Name
	})

	return nil
}

func (in *Files) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*map[string]File)(in)); err != nil {
		return err
	}
	for k, v := range *in {
		// don't set mode for secrets
		if v.Mode == "" && v.Secret.Name == "" {
			v.Mode = guessMode(k)
			(*in)[k] = v
		}
	}
	return nil
}

func (in *File) UnmarshalJSON(data []byte) error {
	if isObject(data) {
		type file File
		return json.Unmarshal(data, (*file)(in))
	} else if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		sec, ok, err := parseSecretReference(s)
		if err != nil {
			return err
		}
		if ok {
			in.Secret = sec.SecretReference
			in.Mode = sec.Mode
		} else {
			in.Content = base64.StdEncoding.EncodeToString([]byte(s))
		}
		return nil
	}

	// assume bytes
	_, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	in.Content = string(data)
	return nil
}

// UnmarshalJSON unmarshalls into this ScopedLabels type from either:
// - a map whose entries look like "containers:foo:key": "v"
// - an array of objects whose entries look like {resourceTYpe: "container" ... value: "v"}
// When unmarshalling from a map, the resulting entries are ordered so that they stay consistent across multiple unmarshallings
func (in *ScopedLabels) UnmarshalJSON(data []byte) error {
	if isObject(data) {
		values := map[string]string{}
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for k, v := range values {
			l, err := parseScopedLabel(k, v)
			if err != nil {
				return err
			}
			*in = append(*in, l)
		}

		sort.Slice(*in, func(i, j int) bool {
			if (*in)[i].ResourceType != (*in)[j].ResourceType {
				return (*in)[i].ResourceType < (*in)[j].ResourceType
			}
			if (*in)[i].ResourceName != (*in)[j].ResourceName {
				return (*in)[i].ResourceName < (*in)[j].ResourceName
			}
			return (*in)[i].Key < (*in)[j].Key
		})
	} else {
		err := json.Unmarshal(data, (*[]ScopedLabel)(in))
		if err != nil {
			return err
		}
		for i, l := range *in {
			nType, err := canonicalResourceType(l.ResourceType)
			if err != nil {
				return err
			}
			(*in)[i].ResourceType = nType
		}
	}

	return nil
}

func (in *CommandSlice) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		parts, err := shlex.Split(s)
		if err != nil {
			return err
		}
		*in = parts
		return nil
	}

	type commandSlice CommandSlice
	return json.Unmarshal(data, (*commandSlice)(in))
}

func (in *AcornBuild) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		in.Context = s
		in.Acornfile = filepath.Join(s, "Acornfile")
		return nil
	}
	type acornBuild AcornBuild
	return json.Unmarshal(data, (*acornBuild)(in))
}

func (in *Build) UnmarshalJSON(data []byte) error {
	if isString(data) {
		s, err := parseString(data)
		if err != nil {
			return err
		}
		in.Context = s
		in.Dockerfile = filepath.Join(s, "Dockerfile")
		return nil
	}
	type build Build
	err := json.Unmarshal(data, (*build)(in))
	if err != nil {
		return err
	}
	if in.Context == "" {
		in.Context = "."
	}
	if in.Dockerfile == "" {
		in.Dockerfile = filepath.Join(in.Context, "Dockerfile")
	}
	return nil
}

func isObject(data []byte) bool {
	return len(data) > 0 && data[0] == '{'
}

func isArray(data []byte) bool {
	return len(data) > 0 && data[0] == '['
}

func parseString(data []byte) (string, error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return "", err
	}
	return str, nil
}

func isString(data []byte) bool {
	return len(data) > 0 && data[0] == '"'
}

type secretReference struct {
	SecretReference
	Mode string
}

func guessMode(s string) string {
	if strings.Contains(s, "/bin/") || strings.Contains(s, "/sbin/") || strings.HasSuffix(s, ".sh") {
		return "0755"
	}
	return "0644"
}

func parseSecretReference(s string) (result secretReference, _ bool, _ error) {
	if strings.HasPrefix(s, "${secret://") && strings.HasSuffix(s, "}") {
		s = s[2 : len(s)-1]
	}

	if !strings.HasPrefix(s, "secret://") {
		return result, false, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return result, false, err
	}

	result.Name = u.Host
	result.Key, _, _ = strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	result.OnChange = ChangeTypeRedeploy

	q := u.Query()
	for k, v := range q {
		for _, v := range v {
			if strings.EqualFold(k, "onchange") {
				if strings.EqualFold(v, "no-action") || strings.EqualFold(v, "noaction") {
					result.OnChange = ChangeTypeNoAction
				}
			} else if strings.EqualFold(k, "mode") {
				_, err := strconv.ParseInt(v, 8, 32)
				if err != nil {
					return result, false, fmt.Errorf("invalid file mode %s: %w", v, err)
				}
				result.Mode = v
			}
		}
	}

	return result, true, nil
}

func parseEnvVar(key, value string) (result EnvVar, _ error) {
	sec, ok, err := parseSecretReference(key)
	if err != nil {
		return result, err
	}
	if ok {
		result.Secret = sec.SecretReference
		return result, nil
	}

	result.Name = key

	sec, ok, err = parseSecretReference(value)
	if err != nil {
		return result, err
	}
	if ok {
		result.Secret = sec.SecretReference
	} else {
		result.Value = value
	}
	return result, nil
}

func parseVolumeDefinition(anonName, s string) (VolumeBinding, error) {
	if s == "" {
		s = "ephemeral://"
	}

	u, err := url.Parse(s)
	if err != nil {
		return VolumeBinding{}, fmt.Errorf("parsing volume reference %s: %w", s, err)
	}

	size := u.Query().Get("size")
	q, err := ParseQuantity(size)
	if err != nil {
		return VolumeBinding{}, err
	}

	result := VolumeBinding{
		Volume:      u.Host,
		Size:        q,
		AccessModes: nil,
	}

	if u.Scheme == "ephemeral" {
		result.Class = u.Scheme
		if result.Volume == "" {
			result.Volume = anonName
		}
	} else if result.Size == "" {
		result.Size = DefaultSizeQuantity
	}

	for _, accessMode := range u.Query()["accessMode"] {
		result.AccessModes = append(result.AccessModes, AccessMode(accessMode))
	}

	for _, accessMode := range u.Query()["accessmode"] {
		result.AccessModes = append(result.AccessModes, AccessMode(accessMode))
	}

	if len(result.AccessModes) == 0 {
		result.AccessModes = AccessModes{AccessModeReadWriteOnce}
	}

	return result, nil
}

func parseVolumeReference(s string) (string, string, error) {
	if !strings.HasPrefix(s, "volume://") && !strings.HasPrefix(s, "ephemeral://") {
		return s, "", nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", "", fmt.Errorf("parsing volume reference %s: %w", s, err)
	}

	subPath := u.Query().Get("subPath")
	if subPath == "" {
		subPath = u.Query().Get("subpath")
	}
	if subPath == "" {
		subPath = u.Query().Get("sub-path")
	}

	return s, subPath, nil
}

func MustParseResourceQuantity(s Quantity) *resource.Quantity {
	if s == "" {
		return nil
	}
	q, err := resource.ParseQuantity(string(s))
	if err != nil {
		panic(fmt.Sprintf("schema did not ensure quantity [%s] was valid: %v", s, err))
	}
	return &q
}

func ParseQuantity(s string) (Quantity, error) {
	if s == "" {
		return "", nil
	}
	_, err := strconv.Atoi(s)
	if err == nil {
		return (Quantity)(s + "G"), nil
	}

	_, err = resource.ParseQuantity(s)
	if err != nil {
		return "", err
	}

	return (Quantity)(s), nil
}

func ParseNameValues(fillEnv bool, s ...string) (result []NameValue) {
	for _, s := range s {
		k, v, _ := strings.Cut(s, "=")
		if v == "" && fillEnv {
			v = os.Getenv(k)
		}
		result = append(result, NameValue{
			Name:  k,
			Value: v,
		})
	}
	return result
}
