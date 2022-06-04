// This file must match its sibling except for the value of index.txt
containers: {
	web: {
		image: "busybox"
		files: {
			"/foo/index.txt": "1"
		}
	}
}
