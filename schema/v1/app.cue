package v1

#AcornBuild: {
	params: [string]: _
	context:   string | *"."
	acornfile: string | *"acorn.cue"
}

#Build: {
	args: [string]: string
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
	alias: string | [...string]
	sidecars: [string]: #Sidecar
}

#Job: {
	#ContainerBase
	schedule: string | *""
	sidecars: [string]: #Sidecar
}

#FileContent: {!~"^secret://"} | {=~"^secret://[a-z][-a-z0-9]*/[a-z][-a-z0-9]*(.optional=true)?$"} | bytes

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
#SecretRef:      =~"^secret://[a-z][-a-z0-9]*(.optional=true)?$"

// The below should work but doesn't. So instead we use the log regexp. This seems like a cue bug
// #Dir: #ShortVolumeRef | #VolumeRef | #EphemeralRef | #ContextDirRef | #SecretRef
#Dir: =~"^[a-z][-a-z0-9]*$|^volume://.+$|^ephemeral://.*$|^$|^\\./.*$|^secret://[a-z][-a-z0-9]*(.optional=true)?$"

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

#SecretOpaque: {
	type:      "opaque"
	optional?: bool
	params?: [string]: _
	data: [string]:    string
}

#SecretTemplate: {
	type:      "template"
	optional?: bool
	data: {
		template: string
	}
}

#SecretToken: {
	type:      "token"
	optional?: bool
	params: {
		characters: string | *"bcdfghjklmnpqrstvwxz2456789"
		length:     (>=0 & <=256) | *54
	}
	data: {
		token?: string
	}
}

#SecretBasicAuth: {
	type:      "basic"
	optional?: bool
	data: {
		username?: string
		password?: string
	}
}

#SecretDocker: {
	type:      "docker"
	optional?: bool
	data: {
		".dockerconfigjson"?: (string | bytes)
	}
}

#SecretSSHAuth: {
	type:      "ssh-auth"
	optional?: bool
	params: {
		algorithm: "rsa" | *"ecdsa"
	}
	data: {
		"ssh-privatekey"?: (string | bytes)
	}
}

#SecretTLS: {
	type:      "tls"
	optional?: bool
	params: {
		algorithm:   "rsa" | *"ecdsa"
		caSecret?:   string
		usage:       *"server" | "client"
		commonName?: string
		organization: [...string]
		sans: [...string]
		durationDays: int | *365
	}
	data: {
		"tls.crt"?: (string | bytes)
		"tls.key"?: (string | bytes)
		"ca.crt"?:  (string | bytes)
		"ca.key"?:  (string | bytes)
	}
}

#SecretGenerated: {
	type:      "generated"
	optional?: bool
	params: {
		job:    string
		format: *"text" | "json"
	}
	data: {}
}

#Secret: *#SecretOpaque | #SecretBasicAuth | #SecretDocker | #SecretSSHAuth | #SecretTLS | #SecretGenerated | #SecretTemplate | #SecretToken

#Acorn: {
	image?:  string
	build?:  string | #AcornBuild
	ports:   #Port | *[...#Port]
	publish: #Port | *[...#Port]
	volumes: [...string]
	secrets: [...string]
	params: [string]: _
}

#App: {
	[=~"params|parameters"]: {
		build: [string]:  _
		deploy: [string]: _
	}
	data: {...}
	containers: [string]: #Container
	jobs: [string]:       #Job
	images: [string]:     #Image
	volumes: [string]:    #Volume
	secrets: [string]:    #Secret
	acorns: [string]:     #Acorn

	_keysMustBeUniqueAcrossTypes: [string]: string
	_keysMustBeUniqueAcrossTypes: {
		for k, v in containers {
			"\(k)": "container"
			if v["alias"] != _|_ {
				if (v.alias & string) != _|_ {
					"\(v.alias)": "alias"
				}
				if !((v.alias & string) != _|_) {
					for alias in v.alias {
						"\(alias)": "alias"
					}
				}
			}
		}
		for k, v in jobs {
			"\(k)": "job"
		}
		for k, v in acorns {
			"\(k)": "acorn"
		}
	}
}
