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
}

#Image: {
	image:  string
	build?: string | *#Build
}

#App: {
	containers: [string]: #Container
	images: [string]:     #Image
}
