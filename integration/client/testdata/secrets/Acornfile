acorns: {
	first: {
		build: {
			acornfile: "first.cue"
		}
		secrets: [
			"parent:first"
		]
	}
	second: {
		build: {
			acornfile: "second.cue"
		}
		secrets: [
			"first.first:second"
		]
	}
}

secrets: parent: {
	data: parent: "true"
}