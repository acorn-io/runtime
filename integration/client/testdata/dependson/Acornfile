containers: {
	one: {
		image: "nginx"
		ports: "80/http"
		dependsOn: ["job1", "job2"]
	}
	two: {
		image: "nginx"
		ports: "80/http"
		dependsOn: "one"
	}
	three: {
		image: "nginx"
		ports: "80/http"
		dependsOn: "two"
	}
}

jobs: {
	job1: {
		image:"busybox"
		command: "/bin/true"
		dependsOn: "job2"
	}
	job2: {
		image:"busybox"
		command: "/bin/true"
	}
}