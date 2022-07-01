package v1

#AcornBuild: {
	buildArgs: [string]: _
	context:   string | *"."
	acornfile: string | *"acorn.cue"
}

#Build: {
	buildArgs: [string]: string
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
	scale?: >=0
	alias?: string
	sidecars: [string]: #Sidecar
}

#Job: {
	#ContainerBase
	schedule: string | *""
	sidecars: [string]: #Sidecar
}

#ProbeMap: {
	[=~"ready|readiness|liveness|startup"]: string | #ProbeSpec
}

#Probes: string | #ProbeMap | [...#ProbeSpec]

#FileContent: {!~"^secret://"} | {=~"^secret://[a-z][-a-z0-9]*/[a-z][-a-z0-9]*(.onchange=(redeploy|no-action)|.mode=[0-7]{3,4})*$"} | bytes | #FileSpec

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
	expose:                         #Port | *[...#Port]
	[=~"probes|probe"]:             #Probes
	[=~"depends[oO]n|depends_on"]:  string | *[...string]
	permissions: {
		rules: [...#RuleSpec]
		clusterRules: [...#RuleSpec]
	}
}

#ShortVolumeRef: =~"^[a-z][-a-z0-9]*$"
#VolumeRef:      =~"^volume://.+$"
#EphemeralRef:   =~"^ephemeral://.*$|^$"
#ContextDirRef:  =~"^\\./.*$"
#SecretRef:      =~"^secret://[a-z][-a-z0-9]*(.onchange=(redeploy|no-action))?$"

// The below should work but doesn't. So instead we use the log regexp. This seems like a cue bug
// #Dir: #ShortVolumeRef | #VolumeRef | #EphemeralRef | #ContextDirRef | #SecretRef
#Dir: =~"^[a-z][-a-z0-9]*$|^volume://.+$|^ephemeral://.*$|^$|^\\./.*$|^secret://[a-z][-a-z0-9]*(.onchange=(redploy|no-action))?$"

#Port: (>0 & <65536) | =~"([0-9]+:)?[0-9]+(/(tcp|udp|http|https))?" | #PortSpec

#AppPort: (>0 & <65536) | =~"([0-9]+:)?[0-9]+(/(tcp|udp|http|https))?" | #AppPortSpec

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
	type: "opaque"
	params?: [string]: _
	data: [string]:    string
}

#SecretTemplate: {
	type: "template"
	data: {
		template: string
	}
}

#SecretToken: {
	type: "token"
	params: {
		// The character set used in the generated string
		characters: string | *"bcdfghjklmnpqrstvwxz2456789"
		// The length of the token to be generated
		length: (>=0 & <=256) | *54
	}
	data: {
		token?: string
	}
}

#SecretBasicAuth: {
	type: "basic"
	data: {
		username?: string
		password?: string
	}
}

#SecretDocker: {
	type: "docker"
	data: {
		".dockerconfigjson"?: (string | bytes)
	}
}

#SecretSSHAuth: {
	type: "ssh-auth"
	params: {
		algorithm: "rsa" | *"ecdsa"
	}
	data: {
		"ssh-privatekey"?: (string | bytes)
	}
}

#SecretTLS: {
	type: "tls"
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
	type: "generated"
	params: {
		job:    string
		format: *"text" | "json"
	}
	data: {}
}

#Secret: *#SecretOpaque | #SecretBasicAuth | #SecretDocker | #SecretSSHAuth | #SecretTLS | #SecretGenerated | #SecretTemplate | #SecretToken

#Acorn: {
	image?: string
	build?: string | #AcornBuild
	ports:  #AppPort | *[...#AppPort]
	expose: #AppPort | *[...#AppPort]
	volumes: [...string]
	secrets: [...string]
	links: [...string]
	deployArgs: [string]: _
	permissions: {
		rules: [...#RuleSpec]
		clusterRules: [...#RuleSpec]
	}
}

#App: {
	args: [string]: _
	profiles: [string]: [string]: _
	[=~"local[dD]ata"]: {...}
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
				"\(v.alias)": "alias"
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
