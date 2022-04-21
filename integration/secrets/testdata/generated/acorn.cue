jobs: {
	pass: {
		env: {
			PASS: "secret://zzz/password"
		}
		image: "busybox"
		cmd: ["sh", "-c", "echo -n $PASS > /run/secrets/output"]
	}
}

secrets: {
	zzz: {
		type: "basic"
		data: {
			password: "static"
		}
	}
	gen: {
		type: "generated"
		params: {
			job: "pass"
		}
	}
}
