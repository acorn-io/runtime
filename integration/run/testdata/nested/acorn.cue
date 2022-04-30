jobs: tester: build: "."
acorns: service: {
	build: "./service"
	ports: "82:81"
}