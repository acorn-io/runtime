containers: default: {
	build: {
		context: "."
	}

	if args.dev {
		build: target: "dynamic"
		expose: "3000/http"
		dirs: "/usr/src": "./"
	}

	if !args.dev {
		build: target: "static"
		expose: "80/http"
	}
}
