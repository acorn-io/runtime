package v1

#Build: {
	context:    string | *"."
	dockerfile: string | *"Dockerfile"
	target:     string | *""
}

#EnvVars: *[...string] | {[string]: string}

#Sidecar: {
	#ContainerBase
	init: bool | *false
}

#Container: {
	#ContainerBase
	sidecars: [string]: #Sidecar
}

#FileContent: {!~"^secret://"} | {=~"^secret://[a-z][-a-z0-9]*/[a-z][-a-z0-9]*$"} | bytes

#ContainerBase: {
	files: [string]:                  #FileContent
	[=~"dirs|directories"]: [string]: #Dir
	// 1 or both of image or build is required
	image?:                         string
	build?:                         string | #Build
	entrypoint:                     string | *[...string]
	[=~"command|cmd"]:              string | *[...string]
	[=~"env|environment"]:          #EnvVars
	[=~"work[dD]ir|working[dD]ir"]: string | *""
	[=~"interactive|tty|stdin"]:    bool | *false
	ports:                          #Port | *[...#Port]
	publish:                        #Port | *[...#Port]
}

#ShortVolumeRef: =~"^[a-z][-a-z0-9]*$"
#VolumeRef:      =~"^volume://.+$"
#EphemeralRef:   =~"^ephemeral://.*$|^$"
#ContextDirRef:  =~"^\\./.*$"
#SecretRef:      =~"^secret://[a-z][-a-z0-9]*$"

// The below should work but doesn't. So instead we use the log regexp. This seems like a cue bug
// #Dir: #ShortVolumeRef | #VolumeRef | #EphemeralRef | #ContextDirRef | #SecretRef
#Dir: =~"^[a-z][-a-z0-9]*$|^volume://.+$|^ephemeral://.*$|^$|^\\./.*$|^secret://[a-z][-a-z0-9]*$"

#Port: (>0 & <65536) | =~"([0-9]+:)?[0-9]+(/(tcp|udp|http|https))?" | #PortSpec

#Image: {
	image:  string
	build?: string | *#Build
}

#Volume: {
	class:       string | *""
	size:        int | *10
	accessModes: [#AccessMode, ...#AccessMode] | #AccessMode | *"readWriteOnce"
}

#App: {
	containers: [string]: #Container
	images: [string]:     #Image
	volumes: [string]:    #Volume
}
