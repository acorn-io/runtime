package v1

#Build: {
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
	target:     string | *""
}

#EnvValue: *[...string] | {[string]: string}

#Sidecar: {
	#ContainerBase
	init: bool | *false
}

#Container: {
	#ContainerBase
	sidecars: [string]: #Sidecar
}

#ContainerBase: {
	files: [string]: string | bytes
	image:      string
	build?:     string | *#Build
	entrypoint: string | *[...string]
	*{
		command: string | *[...string]
	} | {
		cmd: string | *[...string]
	}
	*{
		env: #EnvValue
	} | {
		environment: #EnvValue
	}
	*{
		workdir: string | *""
	} | {
		workingDir: string | *""
	}
	interactive: bool | *false
	ports: [...#Port]
	volumes: [...#VolumeMount]
}

#Port: =~"([0-9]+:)?[0-9]+(/(tcp|udp|http|https))?" | #PortSpec

#VolumeMount: =~"[-a-zA-Z0-9]+:.*" | #VolumeMountSpec

#Image: {
	image:  string
	build?: string | *#Build
}

#Volume: {
	class:      string | *""
	size:       int | *10
	accessMode: [#AccessMode, ...#AccessMode] | #AccessMode | *"readWriteOnce"
}

#App: {
	containers: [string]: #Container
	images: [string]:     #Image
	volumes: [string]:    #Volume
}
