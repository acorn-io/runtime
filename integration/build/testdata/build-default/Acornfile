args: {
	include: bool | *false
}

profiles: build: include: bool | *true

if args.include {
	containers: default: image: "busybox"
}