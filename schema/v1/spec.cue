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

#ContainerBaseSpec: {
	image?: string
	build?: #BuildSpec
	entrypoint: [...string]
	command: [...string]
	environment: [...string]
	workingDir:  string | *""
	interactive: bool | *false
	ports: [...#PortSpec]
	files: [string]: #FileSpec
	dirs: [string]:  #VolumeMountSpec
}

#VolumeMountSpec: {
	{
		volume:  string
		subPath: string | *""
	} |
	{
		contextDir: string
	}
}

#FileSpec: {
	content: string
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

#AppSpec: {
	containers: [string]: #ContainerSpec
	images: [string]:     #ImageSpec
	volumes: [string]:    #VolumeSpec
}

#PortSpec: {
	publish:       bool | *false
	port:          int
	containerPort: int | *port
	protocol:      *"tcp" | "udp" | "http" | "https"
}
