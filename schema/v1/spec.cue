package v1

#SidecarSpec: {
	#ContainerBaseSpec
	init: bool | *false
}

#AcornBuildSpec: {
	context:   string | *"."
	acornfile: string | *"Acornfile"
	buildArgs: [string]: _
}

#BuildSpec: {
	baseImage:  string | *""
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
	target:     string | *""
	contextDirs: [string]: string
	buildArgs: [string]:   string
}

#ProbeSpec: {
	type: *"readiness" | "liveness" | "startup"
	exec?: {
		command: [...string]
	}
	http?: {
		url: string
		headers: [string]: string
	}
	tcp?: {
		url: string
	}
	initialDelaySeconds: uint32 | *0
	timeoutSeconds:      uint32 | *1
	periodSeconds:       uint32 | *10
	successThreshold:    uint32 | *1
	failureThreshold:    uint32 | *3
}

#AliasSpec: {
	name: string
}

#ContainerSpec: {
	#ContainerBaseSpec
	scale?: >=0
	alias?: #AliasSpec
	sidecars: [string]: #SidecarSpec
}

#JobSpec: {
	#ContainerBaseSpec
	schedule: string | *""
	sidecars: [string]: #SidecarSpec
}

#EnvSecretValue: {
	key:      string | *""
	name:     string
	onChange: *"redeploy" | "noAction"
}

#EnvVarSpec: {
	name:    string
	value?:  string
	secret?: #EnvSecretValue
}

#ContainerBaseSpec: {
	image?: string
	build?: #BuildSpec
	entrypoint: [...string]
	command: [...string]
	environment: [...#EnvVarSpec]
	workingDir:  string | *""
	interactive: bool | *false
	ports: [...#PortSpec]
	files: [string]: #FileSpec
	dirs: [string]:  #VolumeMountSpec
	probes: [...#ProbeSpec]
	dependencies: [...#DependencySpec]
	permissions: {
		rules: [...#RuleSpec]
		clusterRules: [...#RuleSpec]
	}
}

#RuleSpec: {
	verbs: [...string]
	apiGroups: [...string]
	resources: [...string]
	resourceNames: [...string]
	nonResourceURLs: [...string]
}

#DependencySpec: {
	targetName: string
}

#VolumeMountSpec: {
	{
		secret: {
			name:     string
			onChange: *"redeploy" | "noAction"
		}
	} |
	{
		volume:  string
		subPath: string | *""
	} |
	{
		contextDir: string
	}
}

#FileSecretSpec: {
	name:     string
	key:      string
	onChange: *"redeploy" | "noAction"
}

#FileSpec: {
	mode: =~"^[0-7]{3,4}$" | *"0644"
	{
		content: string
	} | {
		secret: #FileSecretSpec
	}
}

#ImageSpec: {
	image:  string | *""
	build?: #BuildSpec
}

#AccessMode: "readWriteMany" | "readWriteOnce" | "readOnlyMany"

#VolumeSpec: {
	class:       string | *""
	size:        int | *10
	accessModes: [#AccessMode, ...#AccessMode] | *["readWriteOnce"]
}

#SecretSpec: {
	type: string
	params?: [string]: _
	data: [string]:    (string | bytes)
}

#VolumeBinding: {
	volume:        string
	volumeRequest: string
}

#SecretBinding: {
	secret:        string
	secretRequest: string
}

#ServiceBinding: {
	service: string
	target:  string
}

#AcornSpec: {
	image?: string
	build?: #AcornBuildSpec
	ports: [...#AppPortSpec]
	volumes: [...#VolumeBinding]
	secrets: [...#SecretBinding]
	services: [...#ServiceBinding]
	deployArgs: [string]: _
	permissions: {
		rules: [...#RuleSpec]
		clusterRules: [...#RuleSpec]
	}
}

#AppSpec: {
	containers: [string]: #ContainerSpec
	jobs: [string]:       #JobSpec
	images: [string]:     #ImageSpec
	volumes: [string]:    #VolumeSpec
	secrets: [string]:    #SecretSpec
	acorns: [string]:     #AcornSpec
}

#AppPortSpec: {
	expose:       bool | *false
	port:         int
	internalPort: int | *port
	protocol:     *"tcp" | "udp" | "http" | "https"
}

#PortSpec: {
	expose:       bool | *false
	port:         int
	internalPort: int | *port
	protocol:     *"tcp" | "udp" | "http" | "https"
}
