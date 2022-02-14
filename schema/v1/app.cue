package v1

#Build: {
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
	target:     string | *""
}

#Container: {
	image: string
	build?: string | *#Build
}

#Image: {
	image: string
	build?: string | *#Build
}

#App: {
	containers: [string]: #Container
	images: [string]: #Image
}
