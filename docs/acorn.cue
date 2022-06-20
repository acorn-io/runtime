args: build: {
	liveReload: bool | *false
}

profiles: dev: build: {
	liveReload: bool | *true
}

containers: default: {
	build: {
		context: "."
	}

	if args.build.liveReload {
		build: target: "dynamic"
		expose: "3000/http"
		dirs: "/usr/src": "./"
	}

	if !args.build.liveReload {
		build: target: "static"
		expose: "80/http"
	}
}
