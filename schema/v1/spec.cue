package v1

#SidecarSpec: {
	#ContainerBaseSpec
	init: bool | *false
}

#ContainerSpec: {
	#ContainerBaseSpec
	sidecars: [string]: #SidecarSpec
}

#ContainerBaseSpec: {
	image:  string
	build?: #Build
	entrypoint: [...string]
	command: [...string]
	environment: [...string]
	workingDir:  string | *""
	interactive: bool | *false
	ports: [...#PortSpec]
	files: [string]: #FileSpec
	volumes: [...#VolumeMountSpec]
}

#FileSpec: {
	content: string
}

#ImageSpec: {
	image:  string
	build?: #Build
}

#VolumeMountSpec: {
	volume:    string
	mountPath: string
	subPath:   string | *""
}

#AccessMode: "readWriteMany" | "readWriteOnce" | "readOnlyMany" | "readWriteOncePod"

#VolumeSpec: {
	class:      string | *""
	size:       int | *10
	accessMode: [#AccessMode, ...#AccessMode] | *["readWriteOnce"]
}

#AppSpec: {
	containers: [string]: #ContainerSpec
	images: [string]:     #ImageSpec
	volumes: [string]:    #VolumeSpec
}

#PortSpec: {
	port:          int
	containerPort: int | *port
	protocol:      *"tcp" | "udp" | "http" | "https"
}
