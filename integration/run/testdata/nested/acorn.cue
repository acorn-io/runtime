jobs: tester: build: "."
acorns: service: {
	build: "./service"
	ports: "82:81"
	expose: "83:81"
}