acorns: {
	sub1: {
		build: "./subdir"
	}
	sub2: {
		build: {
			context: "./subdir2"
			acornfile: "./subdir2/test.cue"
			params: {
				filename: "buildfile"
			}
		}
	}
}