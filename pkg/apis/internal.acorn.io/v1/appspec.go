package v1

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	VolumeRequestTypeEphemeral = "ephemeral"

	AccessModeReadWriteMany AccessMode = "readWriteMany"
	AccessModeReadWriteOnce AccessMode = "readWriteOnce"
	AccessModeReadOnlyMany  AccessMode = "readOnlyMany"
)

type AccessMode string

const (
	ChangeTypeRedeploy = "redeploy"
	ChangeTypeNoAction = "noAction"
)

type PathType string

const (
	PathTypeExact  PathType = "exact"
	PathTypePrefix PathType = "prefix"
)

type ChangeType string

type AcornBuild struct {
	OriginalImage string      `json:"originalImage,omitempty"`
	Context       string      `json:"context,omitempty"`
	Acornfile     string      `json:"acornfile,omitempty"`
	BuildArgs     *GenericMap `json:"buildArgs,omitempty"`
}

type Build struct {
	Context            string            `json:"context,omitempty"`
	AdditionalContexts map[string]string `json:"additionalContexts,omitempty"`
	Dockerfile         string            `json:"dockerfile,omitempty"`
	DockerfileContents string            `json:"dockerfileContents,omitempty"`
	Target             string            `json:"target,omitempty"`
	BaseImage          string            `json:"baseImage,omitempty"`
	ContextDirs        map[string]string `json:"contextDirs,omitempty"`
	BuildArgs          map[string]string `json:"buildArgs,omitempty"`
}

func (in Build) BaseBuild() Build {
	return Build{
		Context:    in.Context,
		Dockerfile: in.Dockerfile,
		Target:     in.Target,
	}
}

type Protocol string

var (
	ProtocolTCP  = Protocol("tcp")
	ProtocolUDP  = Protocol("udp")
	ProtocolHTTP = Protocol("http")
)

type PublishProtocol string

var (
	PublishProtocolTCP   = PublishProtocol("tcp")
	PublishProtocolUDP   = PublishProtocol("udp")
	PublishProtocolHTTP  = PublishProtocol("http")
	PublishProtocolHTTPS = PublishProtocol("https")
)

type PortBindings []PortBinding

type PortBinding struct {
	Port     int32    `json:"port,omitempty"`
	Protocol Protocol `json:"protocol,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	// Deprecated Use Hostname instead
	ZZ_ServiceName string `json:"serviceName,omitempty"`
	// Deprecated Has no meaning, publish=true is always assumed
	Publish bool `json:"publish,omitempty"`
	// Deprecated Has no meaning, all ports are exposed by default, if this is true
	// The binding is ignored unless publish is also set to true (which is also deprecated)
	Expose bool `json:"expose,omitempty"`
	// Deprecated All ports are exposed by default
	TargetPort        int32  `json:"targetPort,omitempty"`
	TargetServiceName string `json:"targetServiceName,omitempty"`
}

func (in PortBinding) Complete() PortBinding {
	if in.Hostname != "" && in.Protocol == "" {
		in.Protocol = ProtocolHTTP
	}
	if in.Hostname != "" && in.Protocol != ProtocolHTTP {
		in.Hostname = ""
	}
	return in
}

func (in PortDef) FormatString(serviceName string) string {
	buf := &strings.Builder{}
	if serviceName != "" {
		buf.WriteString(serviceName)
	}
	if in.Port != in.TargetPort {
		if buf.Len() > 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprint(in.Port))
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprint(in.TargetPort))
	buf.WriteString("/")
	buf.WriteString(string(in.Protocol))

	return buf.String()
}

type PortDef struct {
	Hostname   string   `json:"hostname,omitempty"`
	Protocol   Protocol `json:"protocol,omitempty"`
	Publish    bool     `json:"publish,omitempty"`
	Dev        bool     `json:"dev,omitempty"`
	Port       int32    `json:"port,omitempty"`
	TargetPort int32    `json:"targetPort,omitempty"`
}

func (in PortDef) Complete() PortDef {
	if in.TargetPort == 0 {
		in.TargetPort = in.Port
	}
	if in.Port == 0 {
		in.Port = in.TargetPort
	}
	if in.Protocol == "" {
		in.Protocol = ProtocolTCP
	}
	return in
}

type File struct {
	Mode    string          `json:"mode,omitempty"`
	Content string          `json:"content,omitempty"`
	Secret  SecretReference `json:"secret,omitempty"`
}

type VolumeSecretMount struct {
	Name     string     `json:"name,omitempty"`
	OnChange ChangeType `json:"onChange,omitempty"`
}

type VolumeMount struct {
	Volume     string            `json:"volume,omitempty"`
	SubPath    string            `json:"subPath,omitempty"`
	ContextDir string            `json:"contextDir,omitempty"`
	Secret     VolumeSecretMount `json:"secret,omitempty"`
}

type NameValue struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type EnvVar struct {
	Name   string          `json:"name,omitempty"`
	Value  string          `json:"value,omitempty"`
	Secret SecretReference `json:"secret,omitempty"`
}

type SecretReference struct {
	Name     string     `json:"name,omitempty"`
	Key      string     `json:"key,omitempty"`
	OnChange ChangeType `json:"onChange,omitempty"`
}

type Alias struct {
	Name string `json:"name,omitempty"`
}

type ExecProbe struct {
	Command []string `json:"command,omitempty"`
}

type TCPProbe struct {
	URL string `json:"url,omitempty"`
}

type HTTPProbe struct {
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type ProbeType string

const (
	ReadinessProbeType ProbeType = "readiness"
	LivenessProbeType  ProbeType = "liveness"
	StartupProbeType   ProbeType = "startup"
)

type Probe struct {
	Type                ProbeType  `json:"type,omitempty"`
	Exec                *ExecProbe `json:"exec,omitempty"`
	HTTP                *HTTPProbe `json:"http,omitempty"`
	TCP                 *TCPProbe  `json:"tcp,omitempty"`
	InitialDelaySeconds int32      `json:"initialDelaySeconds,omitempty"`
	TimeoutSeconds      int32      `json:"timeoutSeconds,omitempty"`
	PeriodSeconds       int32      `json:"periodSeconds,omitempty"`
	SuccessThreshold    int32      `json:"successThreshold,omitempty"`
	FailureThreshold    int32      `json:"failureThreshold,omitempty"`
}

type Dependency struct {
	TargetName string `json:"targetName,omitempty"`
}

type ScopedLabel struct {
	ResourceType string `json:"resourceType,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
	Key          string `json:"key,omitempty"`
	Value        string `json:"value,omitempty"`
}

type PolicyRule struct {
	rbacv1.PolicyRule `json:",inline"`
	Scopes            []string `json:"scopes,omitempty"`
}

func slicePermutation(in []string) (result [][]string) {
	if len(in) == 0 {
		return [][]string{nil}
	}
	for _, v := range in {
		result = append(result, []string{v})
	}
	return
}

func (p PolicyRule) Exploded() (result []PolicyRule) {
	for _, scope := range slicePermutation(p.Scopes) {
		for _, verb := range slicePermutation(p.Verbs) {
			for _, apiGroup := range slicePermutation(p.APIGroups) {
				for _, resource := range slicePermutation(p.Resources) {
					for _, resourceName := range slicePermutation(p.ResourceNames) {
						for _, nonResourceURL := range slicePermutation(p.NonResourceURLs) {
							result = append(result, PolicyRule{
								PolicyRule: rbacv1.PolicyRule{
									Verbs:           verb,
									APIGroups:       apiGroup,
									Resources:       resource,
									ResourceNames:   resourceName,
									NonResourceURLs: nonResourceURL,
								},
								Scopes: scope,
							})
						}
					}
				}
			}
		}
	}
	return result
}

func matches(allowed, requested []string, emptyAllowedIsAll bool) bool {
	for _, requested := range requested {
		if !matchesSingle(allowed, requested, emptyAllowedIsAll) {
			return false
		}
	}
	return true
}

func matchesSingle(allowed []string, requested string, emptyAllowedIsAll bool) bool {
	if len(requested) == 0 {
		return true
	}
	if emptyAllowedIsAll && len(allowed) == 0 {
		return true
	}
	for _, allow := range allowed {
		if allow == "*" || requested == allow {
			return true
		}
		if strings.HasSuffix(allow, "*") && strings.HasPrefix(requested, allow[:len(allow)-1]) {
			return true
		}
	}

	return false
}

func (p PolicyRule) Grants(currentNamespace string, requested PolicyRule) bool {
	if len(p.NonResourceURLs) > 0 && len(p.Resources) == 0 {
		return len(p.Scopes) == 0 &&
			len(requested.Scopes) == 0 &&
			len(requested.APIGroups) == 0 &&
			len(requested.Resources) == 0 &&
			len(requested.ResourceNames) == 0 &&
			matches(p.NonResourceURLs, requested.NonResourceURLs, false)
	}

	if len(p.NonResourceURLs) > 0 {
		return false
	}

	for _, ns := range p.ResolveNamespaces(currentNamespace) {
		for _, requestedNamespace := range requested.ResolveNamespaces(currentNamespace) {
			if (ns == requestedNamespace || ns == "") &&
				matches(p.Verbs, requested.Verbs, false) &&
				matches(p.APIGroups, requested.APIGroups, false) &&
				matches(p.Resources, requested.Resources, false) &&
				matches(p.ResourceNames, requested.ResourceNames, true) {
				return true
			}
		}
	}

	return false
}

func (p PolicyRule) IsAccountScoped() bool {
	for _, scope := range p.Scopes {
		switch scope {
		case "", "cluster", "account":
			return true
		}
	}
	return false
}

func (p PolicyRule) IsProjectScoped() bool {
	return len(p.Scopes) == 0 || slices.Contains(p.Scopes, "project")
}

func (p PolicyRule) ResolveNamespaces(currentNamespace string) (result []string) {
	namespaces := map[string]struct{}{}
	if p.IsProjectScoped() {
		namespaces[currentNamespace] = struct{}{}
	}
	if p.IsAccountScoped() {
		namespaces[""] = struct{}{}
	}
	for _, namespace := range p.Namespaces() {
		namespaces[namespace] = struct{}{}
	}

	for namespace := range namespaces {
		result = append(result, namespace)
	}

	sort.Strings(result)
	return
}

func (p PolicyRule) Namespaces() (result []string) {
	for _, scope := range p.Scopes {
		if strings.HasPrefix(scope, "namespace:") {
			result = append(result, strings.TrimPrefix(scope, "namespace:"))
		}
		if strings.HasPrefix(scope, "project:") {
			result = append(result, strings.TrimPrefix(scope, "project:"))
		}
	}
	return
}

func GroupByServiceName(perms []Permissions) map[string]Permissions {
	byServiceName := map[string]Permissions{}

	for _, perm := range perms {
		existing := byServiceName[perm.ServiceName]
		existing.ServiceName = perm.ServiceName
		existing.Rules = append(existing.Rules, perm.Rules...)
		byServiceName[perm.ServiceName] = existing
	}

	return byServiceName
}

type Permissions struct {
	ServiceName string       `json:"serviceName,omitempty"`
	Rules       []PolicyRule `json:"rules,omitempty"`
	// Deprecated, use Rules with the 'scopes: ["cluster"]' field
	ZZ_ClusterRules []PolicyRule `json:"clusterRules,omitempty"`
}

func (in Permissions) Grants(currentNamespace string, forService string, requested PolicyRule) bool {
	if in.ServiceName != forService {
		return false
	}

individualRuleLoop:
	for _, individualRule := range requested.Exploded() {
		for _, granted := range in.GetRules() {
			if granted.Grants(currentNamespace, individualRule) {
				continue individualRuleLoop
			}
		}
		return false
	}

	return true
}

func (in Permissions) GetRules() []PolicyRule {
	result := in.Rules
	for _, rule := range in.ZZ_ClusterRules {
		if len(rule.Scopes) == 0 {
			rule.Scopes = append(rule.Scopes, "cluster")
		}
		result = append(result, rule)
	}
	return result
}

func (in *Permissions) HasRules() bool {
	if in == nil {
		return false
	}
	return len(in.Rules) > 0 || len(in.ZZ_ClusterRules) > 0
}

func (in *Permissions) Get() Permissions {
	if in == nil {
		return Permissions{}
	}
	return *in
}

func SimplifySet(perms []Permissions) (result []Permissions) {
	for _, servicePermissions := range GroupByServiceName(perms) {
		result = append(result, Simplify(servicePermissions))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ServiceName < result[j].ServiceName
	})
	return
}

func Simplify(perms Permissions) (result Permissions) {
	var (
		// we sort stuff so don't want to modify the input, thats the reason
		// for the DeepCopy
		rules = perms.DeepCopy().GetRules()
	)

	if len(rules) == 0 {
		return perms
	}

	// This is essentially O(n^2) or maybe worse. So I'm sure there's a better way to do this.
	// This looks for rules where all attributes match except one and then combine the one non-matching
	// attribute
	for {
		var changed bool
		for i := 0; i < len(rules); i++ {
			rule := rules[i]
			tail := rules[i+1:]
			var newTail []PolicyRule
			for _, tailRule := range tail {
				if combined, ok := canCombine(rule, tailRule); ok {
					rule = combined
					changed = true
				} else {
					newTail = append(newTail, tailRule)
				}
			}
			rules = append(rules[0:i], rule)
			rules = append(rules, newTail...)
		}

		if !changed {
			break
		}
	}

	return Permissions{
		ServiceName: perms.ServiceName,
		Rules:       rules,
	}
}

func sortEquals(left, right []string) bool {
	sort.Strings(left)
	sort.Strings(right)
	return slices.Equal(left, right)
}

func combineSlice(left, right []string) (result []string) {
	result = append(result, left...)
	for _, x := range right {
		if !slices.Contains(left, x) {
			result = append(result, x)
		}
	}
	return
}

func canCombine(left, right PolicyRule) (combined PolicyRule, ok bool) {
	var (
		singleDiff bool
	)
	combined = left

	if !sortEquals(combined.Verbs, right.Verbs) {
		if singleDiff {
			return combined, false
		}
		combined.Verbs = combineSlice(combined.Verbs, right.Verbs)
		singleDiff = true
	}

	if !sortEquals(combined.APIGroups, right.APIGroups) {
		if singleDiff {
			return combined, false
		}
		combined.APIGroups = combineSlice(combined.APIGroups, right.APIGroups)
		singleDiff = true
	}

	if !sortEquals(combined.Resources, right.Resources) {
		if singleDiff {
			return combined, false
		}
		combined.Resources = combineSlice(combined.Resources, right.Resources)
		singleDiff = true
	}

	if !sortEquals(combined.ResourceNames, right.ResourceNames) {
		if singleDiff {
			return combined, false
		}
		combined.ResourceNames = combineSlice(combined.ResourceNames, right.ResourceNames)
		singleDiff = true
	}

	if !sortEquals(combined.NonResourceURLs, right.NonResourceURLs) {
		if singleDiff {
			return combined, false
		}
		combined.NonResourceURLs = combineSlice(combined.NonResourceURLs, right.NonResourceURLs)
		singleDiff = true
	}

	if !sortEquals(combined.Scopes, right.Scopes) {
		if singleDiff {
			return combined, false
		}
		combined.Scopes = combineSlice(combined.Scopes, right.Scopes)
		singleDiff = true
	}

	return combined, singleDiff
}

func GrantsAll(currentNamespace string, requestedPerms, grantedPerms []Permissions) (missing []Permissions, granted bool) {
	var (
		grantedByServiceName = GroupByServiceName(grantedPerms)
	)

	for serviceName, requestedPerm := range GroupByServiceName(requestedPerms) {
		if subMissing, subGranted := Grants(currentNamespace, requestedPerm, grantedByServiceName[serviceName]); !subGranted {
			missing = append(missing, Simplify(subMissing))
		}
	}

	return missing, len(missing) == 0
}

func Grants(currentNamespace string, requestedPermissions, grantedPermissions Permissions) (missing Permissions, granted bool) {
	missing.ServiceName = requestedPermissions.ServiceName

	for _, requested := range requestedPermissions.Rules {
		if grantedPermissions.Grants(currentNamespace, requestedPermissions.ServiceName, requested) {
			continue
		}
		missing.Rules = append(missing.Rules, requested)
	}

	return Simplify(missing), len(missing.Rules) == 0
}

func FindPermission(serviceName string, perms []Permissions) (result Permissions) {
	result.ServiceName = serviceName
	for _, perm := range perms {
		if perm.ServiceName == serviceName {
			result.Rules = append(result.Rules, perm.GetRules()...)
		}
	}
	return
}

type Files map[string]File

type CommandSlice []string

type EnvVars []EnvVar

type NameValues []NameValue

type Probes []Probe

type Ports []PortDef

type Dependencies []Dependency

type ScopedLabels []ScopedLabel

type Container struct {
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Dirs         map[string]VolumeMount `json:"dirs,omitempty"`
	Files        Files                  `json:"files,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Build        *Build                 `json:"build,omitempty"`
	Command      CommandSlice           `json:"command,omitempty"`
	Interactive  bool                   `json:"interactive,omitempty"`
	Entrypoint   CommandSlice           `json:"entrypoint,omitempty"`
	Environment  EnvVars                `json:"environment,omitempty"`
	WorkingDir   string                 `json:"workingDir,omitempty"`
	Ports        Ports                  `json:"ports,omitempty"`
	Probes       Probes                 `json:"probes"` // Don't omitempty so that nil vs empty is recorded
	Dependencies Dependencies           `json:"dependencies,omitempty"`
	Permissions  *Permissions           `json:"permissions,omitempty"`
	ComputeClass *string                `json:"class,omitempty"`
	Memory       *int64                 `json:"memory,omitempty"`

	// Metrics is available on containers and jobs, but not sidecars
	Metrics MetricsDef `json:"metrics,omitempty"`

	// Scale is only available on containers, not sidecars or jobs
	Scale *int32 `json:"scale,omitempty"`

	// Schedule is only available on jobs
	Schedule string `json:"schedule,omitempty"`

	// Events is only available on jobs
	Events []string `json:"events,omitempty"`

	// Init is only available on sidecars
	Init bool `json:"init,omitempty"`

	// Sidecars are not available on sidecars
	Sidecars map[string]Container `json:"sidecars,omitempty"`
}

type Image struct {
	Image      string      `json:"image,omitempty"`
	Build      *Build      `json:"containerBuild,omitempty"`
	AcornBuild *AcornBuild `json:"build,omitempty"`
}

type AppSpec struct {
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Readme      string                   `json:"readme,omitempty"`
	Info        string                   `json:"info,omitempty"`
	Icon        string                   `json:"icon,omitempty"`
	Containers  map[string]Container     `json:"containers,omitempty"`
	Jobs        map[string]Container     `json:"jobs,omitempty"`
	Images      map[string]Image         `json:"images,omitempty"`
	Volumes     map[string]VolumeRequest `json:"volumes,omitempty"`
	Secrets     map[string]Secret        `json:"secrets,omitempty"`
	Acorns      map[string]Acorn         `json:"acorns,omitempty"`
	Routers     map[string]Router        `json:"routers,omitempty"`
	Services    map[string]Service       `json:"services,omitempty"`
}

type Route struct {
	Path              string   `json:"path,omitempty"`
	TargetServiceName string   `json:"targetServiceName,omitempty"`
	TargetPort        int      `json:"targetPort,omitempty"`
	PathType          PathType `json:"pathType,omitempty"`
}

type Routes []Route

type Router struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Routes      Routes            `json:"routes,omitempty"`
}

func (in Acorn) GetOriginalImage() string {
	originalImage := in.Image
	if in.Build != nil && in.Build.OriginalImage != "" {
		originalImage = in.Build.OriginalImage
	}
	return originalImage
}

type Acorn struct {
	Labels              ScopedLabels           `json:"labels,omitempty"`
	Annotations         ScopedLabels           `json:"annotations,omitempty"`
	Name                string                 `json:"name,omitempty"`
	Description         string                 `json:"description,omitempty"`
	Image               string                 `json:"image,omitempty"`
	Build               *AcornBuild            `json:"build,omitempty"`
	Profiles            []string               `json:"profiles,omitempty"`
	DeployArgs          *GenericMap            `json:"deployArgs,omitempty"`
	Publish             PortBindings           `json:"publish,omitempty"`
	PublishMode         PublishMode            `json:"publishMode,omitempty"`
	Environment         NameValues             `json:"environment,omitempty"`
	Secrets             SecretBindings         `json:"secrets,omitempty"`
	Volumes             VolumeBindings         `json:"volumes,omitempty"`
	Links               ServiceBindings        `json:"links,omitempty"`
	AutoUpgrade         *bool                  `json:"autoUpgrade,omitempty"`
	NotifyUpgrade       *bool                  `json:"notifyUpgrade,omitempty"`
	AutoUpgradeInterval string                 `json:"autoUpgradeInterval,omitempty"`
	Memory              MemoryMap              `json:"memory,omitempty"`
	ComputeClasses      ComputeClassMap        `json:"class,omitempty"`
	Permissions         map[string]Permissions `json:"permissions,omitempty"`
}

type Secret struct {
	External    string            `json:"external,omitempty"`
	Alias       string            `json:"alias,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type,omitempty"`
	Params      *GenericMap       `json:"params,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
}

type AccessModes []AccessMode

type VolumeRequest struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Class       string            `json:"class,omitempty"`
	Size        Quantity          `json:"size,omitempty"`
	AccessModes AccessModes       `json:"accessModes,omitempty"`
}

// Workload to its memory
type MemoryMap map[string]*int64

// Workload to its class
type ComputeClassMap map[string]string

type MetricsDef struct {
	Port int32  `json:"port,omitempty"`
	Path string `json:"path,omitempty"`
}

type GeneratedService struct {
	Job string `json:"job,omitempty"`
}

type Service struct {
	Labels              ScopedLabels           `json:"labels,omitempty"`
	Annotations         ScopedLabels           `json:"annotations,omitempty"`
	Name                string                 `json:"name,omitempty"`
	Description         string                 `json:"description,omitempty"`
	Default             bool                   `json:"default,omitempty"`
	External            string                 `json:"external,omitempty"`
	Alias               string                 `json:"alias,omitempty"`
	Address             string                 `json:"address,omitempty"`
	Ports               Ports                  `json:"ports,omitempty"`
	Container           string                 `json:"container,omitempty"`
	Data                *GenericMap            `json:"data,omitempty"`
	Generated           *GeneratedService      `json:"generated,omitempty"`
	Image               string                 `json:"image,omitempty"`
	Build               *AcornBuild            `json:"build,omitempty"`
	ServiceArgs         *GenericMap            `json:"serviceArgs,omitempty"`
	Environment         NameValues             `json:"environment,omitempty"`
	Secrets             SecretBindings         `json:"secrets,omitempty"`
	Links               ServiceBindings        `json:"links,omitempty"`
	AutoUpgrade         *bool                  `json:"autoUpgrade,omitempty"`
	NotifyUpgrade       *bool                  `json:"notifyUpgrade,omitempty"`
	AutoUpgradeInterval string                 `json:"autoUpgradeInterval,omitempty"`
	Memory              MemoryMap              `json:"memory,omitempty"`
	ComputeClasses      ComputeClassMap        `json:"class,omitempty"`
	Permissions         map[string]Permissions `json:"permissions,omitempty"`
	Consumer            *ServiceConsumer       `json:"consumer,omitempty"`
	ContainerLabels     map[string]string      `json:"containerLabels,omitempty"`
}

func (in Service) GetOriginalImage() string {
	originalImage := in.Image
	if in.Build != nil && in.Build.OriginalImage != "" {
		originalImage = in.Build.OriginalImage
	}
	return originalImage
}

func (in Service) GetJob() string {
	if in.Generated == nil {
		return ""
	}
	return in.Generated.Job
}
