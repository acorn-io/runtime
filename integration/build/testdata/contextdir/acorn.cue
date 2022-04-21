containers: {
	simple: {
		build: {
		    target: "not-default"
		}
        dirs: "/var/lib/www": "./files"
	}
	fromImage: {
        image: "nginx"
        dirs: "/var/lib/www": "./files"
	}
}