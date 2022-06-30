containers: {
	level2: image: "nginx"
	level2: dirs: "/asdf": "vol"
	level2: sidecars: side2: image: "busybox"
	level2: sidecars: side2: command: "sleep 3600"
}

acorns: {
	level3: {
		build: {
			acornfile: "level3.cue"
		}
	}
}
