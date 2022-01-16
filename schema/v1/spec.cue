package v1

#ContainerSpec: {
	image: string
	build: #Build
}

#AppSpec: {
	containers: [string]: #ContainerSpec
}
