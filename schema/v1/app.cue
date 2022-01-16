package v1

#Build: {
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
}

#Container: {
	image: string
	build: string | *#Build
}

#App: {
	containers: [string]: #Container
}
