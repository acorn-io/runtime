containers: {
	level2: image: "nginx"
	level2: dirs: "/var": "vol"
	level2: sidecars: side2: image: "nginx"
}

acorns: {
	level3: {
		build: {
			acornfile: "level3.cue"
		}
	}
}
