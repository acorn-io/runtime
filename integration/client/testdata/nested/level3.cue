containers: {
	level3: image: "nginx"
	level3: dirs: "/asdf": "vol"
	level3: sidecars: side3: image: "busybox"
	level3: sidecars: side3: command: "sleep 3600"
}
