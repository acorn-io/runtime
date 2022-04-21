params: {
	deploy: {
		someInt: int
	}
}

containers: {
	foo: {
		env: {
			arg: "\(params.deploy.someInt)"
		}
		image: "nginx"
	}
}
