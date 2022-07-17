jobs: {
	pass: {
		env: {
			PASS: "secret://zzz/password"
		}
		image: "busybox"
		cmd: ["sh", "-c", "echo -n $PASS > /run/secrets/output"]
	}
	cronpass: {schedule: "* * * * * "} & pass
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
	gen2: {
		type: "generated"
		params: job: "cronpass"
	}
}
