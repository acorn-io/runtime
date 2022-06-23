containers: {
	level3: image: "nginx"
	level3: dirs: "/var": "vol"
	level3: sidecars: side3: image: "nginx"
}
