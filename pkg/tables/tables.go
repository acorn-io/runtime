package tables

var (
	CheckResult = [][]string{
		{"Name", "Name"},
		{"Passed", "Passed"},
		{"Message", "Message"},
	}

	App = [][]string{
		{"Name", "{{ . | name }}"},
		{"Image", "{{ trunc .Status.AppImage.Name }}"},
		{"Healthy", "Status.Columns.Healthy"},
		{"Up-To-Date", "Status.Columns.UpToDate"},
		{"Created", "{{ago .CreationTimestamp}}"},
		{"Endpoints", "Status.Columns.Endpoints"},
		{"Message", "{{ appGeneration . .Status.Columns.Message }}"},
	}
	AppConverter = MustConverter(App)

	Volume = [][]string{
		{"Name", "{{ . | name }}"},
		{"App-Name", "Status.AppPublicName"},
		{"Bound-Volume", "Status.VolumeName"},
		{"Capacity", "Spec.Capacity"},
		{"Volume-Class", "{{ .Spec.Class }}"},
		{"Status", "Status.Status"},
		{"Access-Modes", "Status.Columns.AccessModes"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	VolumeConverter = MustConverter(Volume)

	VolumeClass = [][]string{
		{"Name", "{{ . | name }}"},
		{"Default", "{{ boolToStar .Default }}"},
		{"Inactive", "{{ boolToStar .Inactive }}"},
		{"Storage-Class", "{{ .StorageClassName }}"},
		{"Size-Range", "{{ displayRange .Size.Min .Size.Max }}"},
		{"Default-Size", "{{ .Size.Default }}"},
		{"Access-Modes", "{{ pointer .AllowedAccessModes }}"},
		{"Regions", "{{ arrayNoSpace .SupportedRegions }}"},
		{"Description", "{{ .Description }}"},
	}
	VolumeClassConverter = MustConverter(VolumeClass)

	Service = [][]string{
		{"Name", "{{ . | name }}"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	ServiceConverter = MustConverter(Service)

	// Used for acorn image related printing
	ImageAcorn = [][]string{
		{"Repository", "{{if eq .Repository \"\"}}<none>{{else}}{{.Repository}}{{end}}"},
		{"Tag", "{{if eq .Tag \"\"}}<none>{{else}}{{.Tag}}{{end}}"},
		{"Image-ID", "{{trunc .Name}}"},
	}

	// Used for kubectl image related printing
	Image = [][]string{
		{"Image-ID", "{{trunc .Name}}"},
		{"Tags", "{{if .Tags}}{{else}}<none>{{end}}{{range $index, $v := .Tags}}{{if $index}},{{end}}{{if eq $v \"\"}}<none>{{else}}{{$v}}{{end}}{{end}}"},
	}
	ImageConverter = MustConverter(Image)

	ImageContainer = [][]string{
		{"Repository", "{{ .Repo }}"},
		{"Tag", "{{ .Tag }}"},
		{"Image-ID", "{{trunc .ImageID }}"},
		{"Container", "{{ .Container}}"},
		{"Digest", "{{ .Digest }}"},
	}
	ImageContainerConverter = MustConverter(ImageContainer)

	Container = [][]string{
		{"Name", "{{ . | name }}"},
		{"App", "Status.Columns.App"},
		{"Image", "Spec.Image"},
		{"State", "Status.Columns.State"},
		{"RestartCount", "Status.RestartCount"},
		{"Created", "{{ago .CreationTimestamp}}"},
		{"Message", "Status.PodMessage"},
	}
	ContainerConverter = MustConverter(Container)

	CredentialClient = [][]string{
		{"Server", "ServerAddress"},
		{"Username", "Username"},
		{"Local", "{{boolToStar .LocalStorage}}"},
	}

	Credential = [][]string{
		{"Server", "ServerAddress"},
		{"Username", "Username"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	CredentialConverter = MustConverter(Credential)

	Secret = [][]string{
		{"Name", "{{ . | name }}"},
		{"Type", "Type"},
		{"Keys", "Keys"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	SecretConverter = MustConverter(Secret)

	Info = [][]string{
		{"Version", "Client.Version"},
		{"Current Project", "Client.CLI.CurrentProject"},
		{"Hub Servers", "{{ arrayNoSpace .Client.CLI.HubServers }}"},
	}
	InfoConverter = MustConverter(Info)

	Builder = [][]string{
		{"Name", "Name"},
		{"Ready", "Status.Ready"},
	}
	BuilderConverter = MustConverter(Builder)

	ComputeClass = [][]string{
		{"Name", "Name"},
		{"Default", "{{ boolToStar .Default }}"},
		{"Memory Range", "{{ memoryToRange .Memory }}"},
		{"Memory Default", "{{ defaultMemory .Memory }}"},
		{"Regions", "{{ arrayNoSpace .SupportedRegions }}"},
		{"Description", "Description"},
	}
	ComputeClassConverter = MustConverter(ComputeClass)

	Build = [][]string{
		{"Name", "Name"},
		{"Image", "Status.AppImage.ID"},
		{"Message", "Status.BuildError"},
	}
	BuildConverter = MustConverter(Build)

	ImageAllowRule = [][]string{
		{"Name", "{{ . | name }}"},
	}
	ImageAllowRuleConverter = MustConverter(ImageAllowRule)

	Project = [][]string{
		{"Name", "Name"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	ProjectConverter = MustConverter(Project)

	ProjectClient = [][]string{
		{"Name", "Name"},
		{"Default", "{{ boolToStar .Default }}"},
		{"Regions", "{{ arrayNoSpace .Regions }}"},
	}

	Region = [][]string{
		{"Name", "Name"},
		{"Account", "{{ ownerName . }}"},
		{"Region Name", "{{ .Spec.RegionName }}"},
		{"Created", "{{ ago .CreationTimestamp }}"},
		{"Description", "{{ .Spec.Description }}"},
	}
	RegionConverter = MustConverter(Region)

	RuleRequests = [][]string{
		{"Service", "Service"},
		{"Verbs", "Verbs"},
		{"Namespace", "Namespace"},
		{"Resource", "Resource"},
		{"Scope", "Scope"},
	}

	Event = [][]string{
		{"Name", "{{ . | name }}"},
		{"Type", "Type"},
		{"Actor", "Actor"},
		{"Source", "Source"},
		{"Observed", "{{ ago .Observed }}"},
		{"Description", "Description"},
	}

	EventConverter = MustConverter(Event)
)
