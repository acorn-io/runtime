package tables

var (
	CheckResult = [][]string{
		{"Name", "Name"},
		{"Passed", "Passed"},
		{"Message", "Message"},
	}

	CheckResultConverter = MustConverter(CheckResult)

	App = [][]string{
		{"Name", "{{ . | name }}"},
		{"Image", "{{ trunc .Spec.Image }}"},
		{"Healthy", "Status.Columns.Healthy"},
		{"Up-To-Date", "Status.Columns.UpToDate"},
		{"Created", "{{ago .CreationTimestamp}}"},
		{"Endpoints", "Status.Columns.Endpoints"},
		{"Message", "{{ appGeneration . .Status.Columns.Message }}"},
	}
	AppConverter = MustConverter(App)

	Volume = [][]string{
		{"Name", "{{ . | name }}"},
		{"App-Name", "Status.AppName"},
		{"Bound-Volume", "Status.VolumeName"},
		{"Capacity", "Spec.Capacity"},
		{"Status", "Status.Status"},
		{"Access-Modes", "Status.Columns.AccessModes"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	VolumeConverter = MustConverter(Volume)

	Image = [][]string{
		{"Repository", "{{if eq .Repository \"\"}}<none>{{else}}{{.Repository}}{{end}}"},
		{"Tag", "{{if eq .Tag \"\"}}<none>{{else}}{{.Tag}}{{end}}"},
		{"Image-ID", "{{trunc .Name}}"},
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

	Credential = [][]string{
		{"Server", "ServerAddress"},
		{"Username", "Username"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	CredentialConverter = MustConverter(Credential)

	Secret = [][]string{
		{"Alias", "{{alias .}}"},
		{"Name", "{{ . | name }}"},
		{"Type", "Type"},
		{"Keys", "Keys"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	SecretConverter = MustConverter(Secret)

	Info = [][]string{
		{"Version", "Version"},
		{"Controller-Image", "ControllerImage"},
	}
	InfoConverter = MustConverter(Info)

	Builder = [][]string{
		{"Name", "Name"},
		{"Ready", "Status.Ready"},
	}
	BuilderConverter = MustConverter(Builder)

	RuleRequests = [][]string{
		{"Service", "Service"},
		{"Verbs", "Verbs"},
		{"Resource", "Resource"},
		{"Scope", "Scope"},
	}
)
