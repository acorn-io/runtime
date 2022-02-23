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
}

#ImageSpec: {
	image:  string
	build?: #Build
}

#AppSpec: {
	containers: [string]: #ContainerSpec
	images: [string]:     #ImageSpec
}
