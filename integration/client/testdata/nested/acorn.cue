containers: {
	level1: image: "nginx"
	level1: dirs: "/var": "vol"
	level1: sidecars: side1: image: "nginx"
}

acorns: {
	level2: {
		build: {
			acornfile: "level2.cue"
		}
	}
}
