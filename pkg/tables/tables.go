package tables

var (
	CheckResult = [][]string{
		{"Name", "Name"},
		{"Passed", "Passed"},
		{"Message", "Message"},
	}

	App = [][]string{
		{"Name", "{{ . | name }}"},
		{"Image", "{{ . | imageName | trunc }}"},
		{"Commit", "{{ . | imageCommit | trunc }}"},
		{"Created", "{{ago .CreationTimestamp}}"},
		{"Endpoints", "Status.Columns.Endpoints"},
		{"Message", "{{ appGeneration . .Status.Columns.Message }}"},
	}
	AppConverter = MustConverter(App)

	Volume = [][]string{
		{"Name", "{{ . | name }}"},
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
		{"Acorn", "Status.Columns.App"},
		{"Image", "Spec.Image"},
		{"State", "Status.Columns.State"},
		{"RestartCount", "Status.RestartCount"},
		{"Created", "{{ago .CreationTimestamp}}"},
		{"Message", "Status.PodMessage"},
	}
	ContainerConverter = MustConverter(Container)

	Job = [][]string{
		{"Name", "{{ . | name }}"},
		{"State", "Status.State"},
		{"Last Run", "{{lastRun .Status.LastRun }}"},
		{"Next Run", "{{nextRun .Status.NextRun }}"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	JobConverter = MustConverter(Job)

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
		{"Manager Servers", "{{ arrayNoSpace .Client.CLI.AcornServers }}"},
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

	ImageRoleAuthorization = [][]string{
		{"Name", "{{ . | name }}"},
	}
	ImageRoleAuthorizationConverter = MustConverter(ImageRoleAuthorization)

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
		{"Verbs/Actions", "Verbs"},
		{"Resources/API", "Resource"},
		{"Scope", "Scope"},
	}

	Event = [][]string{
		{"Resource", "Resource"},
		{"Name", "{{ . | name }}"},
		{"Type", "Type"},
		{"Actor", "Actor"},
		{"Observed", "Observed"},
		{"Description", "Description"},
	}

	EventConverter = MustConverter(Event)
)
