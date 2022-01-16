containers: {
	one: {
		build: "one"
	}
	two: {
		build: {
			dockerfile: "two/subdir/Dockerfile.txt"
			context: "two/subdir/subdir2"
		}
	}
}
