package v1

#ContainerSpec: {
	image: string
	build?: #Build
}

#ImageSpec: {
	image: string
	build?: #Build
}

#AppSpec: {
	containers: [string]: #ContainerSpec
	images: [string]: #ImageSpec
}
