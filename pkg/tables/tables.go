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
		{"App-Name", "Status.AppName"},
		{"Bound-Volume", "Status.VolumeName"},
		{"Capacity", "Spec.Capacity"},
		{"Status", "Status.Status"},
		{"Access-Modes", "Status.Columns.AccessModes"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	VolumeConverter = MustConverter(Volume)

	//Used for acorn image related printing
	ImageAcorn = [][]string{
		{"Repository", "{{if eq .Repository \"\"}}<none>{{else}}{{.Repository}}{{end}}"},
		{"Tag", "{{if eq .Tag \"\"}}<none>{{else}}{{.Tag}}{{end}}"},
		{"Image-ID", "{{trunc .Name}}"},
	}

	//Used for kubectl image related printing
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

	Build = [][]string{
		{"Name", "Name"},
		{"Image", "Status.AppImage.ID"},
		{"Message", "Status.BuildError"},
	}
	BuildConverter = MustConverter(Build)

	Project = [][]string{
		{"Name", "Name"},
		{"Created", "{{ago .CreationTimestamp}}"},
	}
	ProjectConverter = MustConverter(Project)

	RuleRequests = [][]string{
		{"Service", "Service"},
		{"Verbs", "Verbs"},
		{"Namespace", "Namespace"},
		{"Resource", "Resource"},
		{"Scope", "Scope"},
	}
)
