package v1

#SidecarSpec: {
	#ContainerBaseSpec
	init: bool | *false
}

#BuildSpec: {
	baseImage:  string | *""
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
	target:     string | *""
	contextDirs: [string]: string
}

#ContainerSpec: {
	#ContainerBaseSpec
	sidecars: [string]: #SidecarSpec
}

#EnvSecretValue: {
	key:       string | *""
	name:      string
	optional?: bool
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
}

#VolumeMountSpec: {
	{
		secret: {
			name:      string
			optional?: bool
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
	name:      string
	key:       string
	optional?: bool
}

#FileSpec: {
	{
		content: string
	} | {
		secret: #FileSecretSpec
	}
}

#ImageSpec: {
	image:  string
	build?: #BuildSpec
}

#AccessMode: "readWriteMany" | "readWriteOnce" | "readOnlyMany" | "readWriteOncePod"

#VolumeSpec: {
	class:       string | *""
	size:        int | *10
	accessModes: [#AccessMode, ...#AccessMode] | *["readWriteOnce"]
}

#SecretSpec: {
	type:      string
	optional?: bool
	params?: [string]: _
	data: [string]:    (string | bytes)
}

#AppSpec: {
	containers: [string]: #ContainerSpec
	images: [string]:     #ImageSpec
	volumes: [string]:    #VolumeSpec
	secrets: [string]:    #SecretSpec
}

#PortSpec: {
	publish:       bool | *false
	port:          int
	containerPort: int | *port
	protocol:      *"tcp" | "udp" | "http" | "https"
}
