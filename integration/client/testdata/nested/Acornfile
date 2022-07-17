containers: {
	level1: image: "nginx"
	level1: dirs: "/asdf": "vol"
	level1: sidecars: side1: image: "busybox"
	level1: sidecars: side1: command: "sleep 3600"
}

acorns: {
	level2: {
		build: {
			acornfile: "level2.cue"
		}
	}
}
