containers: {
	one: {
		build: "one"
	}
	two: {
	    build: images.itwo.build
	}
	three: {
	    image: "busybox"
	}
}

images: {
	ione: {
		build: "one"
	}
	itwo: {
		build: {
			dockerfile: "two/subdir/Dockerfile.txt"
			context: "two/subdir/subdir2"
		}
	}
	three: {
	    image: "busybox"
	}
}
