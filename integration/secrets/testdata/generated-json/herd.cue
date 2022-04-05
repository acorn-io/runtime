jobs: {
	pass: {
		env: {
			PASS: "secret://zzz/password"
		}
		image: "busybox"
		files: "/run.sh": """
			#!/bin/sh
			cat << EOF > /run/secrets/output
			{
			    "type": "other",
			    "data": {
			        "key": "value",
			        "pass": "$PASS"
			    }
			}
			EOF
			"""
		cmd: ["sh", "/run.sh"]
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
			format: "json"
			job:    "pass"
		}
	}
}
