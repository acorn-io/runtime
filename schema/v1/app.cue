package v1

#Build: {
	buildArgs: [string]: string
	context:    string | *"."
	dockerfile: string | *""
	target:     string | *""
}

#EnvVars: *[...string] | {[string]: string}

#Sidecar: {
	#ContainerBase
	init: bool | *false
}

#Container: {
	#ContainerBase
	labels:                       [string]: string
	annotations:                  [string]: string
	scale?: >=0
	sidecars: [string]: #Sidecar
}

#Job: {
	#ContainerBase
	labels:                       [string]: string
	annotations:                  [string]: string
	schedule: string | *""
	sidecars: [string]: #Sidecar
}

#ProbeMap: {
	[=~"ready|readiness|liveness|startup"]: string | #ProbeSpec
}

#PortMap: {
	internal: #PortSingle | *[...#Port]
	expose:   #PortSingle | *[...#Port]
	publish:  #PortSingle | *[...#Port]
}

#ProbeSpec: {
	type: *"readiness" | "liveness" | "startup"
	exec?: {
		command: [...string]
	}
	http?: {
		url: string
		headers: [string]: string
	}
	tcp?: {
		url: string
	}
	initialDelaySeconds: uint32 | *0
	timeoutSeconds:      uint32 | *1
	periodSeconds:       uint32 | *10
	successThreshold:    uint32 | *1
	failureThreshold:    uint32 | *3
}

#Probes: string | #ProbeMap | [...#ProbeSpec]

#FileSecretSpec: {
	name:     string
	key:      string
	onChange: *"redeploy" | "noAction"
}

#FileSpec: {
	mode: =~"^[0-7]{3,4}$" | *"0644"
	{
		content: string
	} | {
		secret: #FileSecretSpec
	}
}

#FileContent: {!~"^secret://"} | {=~"^secret://[a-z][-a-z0-9]*/[a-z][-a-z0-9]*(.onchange=(redeploy|no-action)|.mode=[0-7]{3,4})*$"} | #FileSpec

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
	ports:                          #PortSingle | *[...#Port] | #PortMap
	[=~"probes|probe"]:             #Probes
	[=~"depends[oO]n|depends_on"]:  string | *[...string]
	permissions: {
		rules: [...#RuleSpec]
		clusterRules: [...#ClusterRuleSpec]
	}
}

#ShortVolumeRef: "^[a-z][-a-z0-9]*$"
#VolumeRef:      "^volume://.+$"
#EphemeralRef:   "^ephemeral://.*$|^$"
#ContextDirRef:  "^\\./.*$"
#SecretRef:      "^secret://[a-z][-a-z0-9]*(.onchange=(redeploy|no-action))?$"

// The below should work but doesn't. So instead we use the log regexp. This seems like a cue bug
// #Dir: #ShortVolumeRef | #VolumeRef | #EphemeralRef | #ContextDirRef | #SecretRef
#Dir: =~"^[a-z][-a-z0-9]*$|^volume://.+$|^ephemeral://.*$|^$|^\\./.*$|^secret://[a-z][-a-z0-9]*(.onchange=(redeploy|no-action))?$"

#PortSingle: (>0 & <65536) | =~#PortRegexp
#Port:       (>0 & <65536) | =~#PortRegexp | #PortSpec
#PortRegexp: #"^([a-z][-a-z0-9]+:)?([0-9]+:)?([a-z][-a-z0-9]+:)?([0-9]+)(/(tcp|udp|http))?$"#

#PortSpec: {
	publish:           bool | *false
	expose:            bool | *false
	port:              int | *targetPort
	targetPort:        int
	targetServiceName: string | *""
	serviceName:       string | *""
	protocol:          *"" | "tcp" | "udp" | "http"
}

// Allowing [resourceType:][resourceName:][some.random/key]
#ScopedLabelMapKey: =~"^([a-z][-a-z0-9]+:)?([a-z][-a-z0-9]+:)?([a-z][-a-z0-9./]+)?$"
#ScopedLabelMap: {[#ScopedLabelMapKey]: string}
#ScopedLabel: {
	resourceType: =~#DNSName | *""
	resourceName: =~#DNSName | *""
	key:          =~"[a-z][-a-z0-9./][a-z]*"
	value:        string | *""
}


#RuleSpec: {
	verbs: [...string]
	apiGroups: [...string]
	resources: [...string]
	resourceNames: [...string]
	nonResourceURLs: [...string]
} | string

#ClusterRuleSpec: {
	verbs: [...string]
	namespaces: [...string]
	apiGroups: [...string]
	resources: [...string]
	resourceNames: [...string]
	nonResourceURLs: [...string]
} | string

#Image: {
	image:  string | *""
	build?: string | *#Build
}

#AccessMode: "readWriteMany" | "readWriteOnce" | "readOnlyMany"

#Volume: {
	labels:      [string]: string
	annotations: [string]: string
	class:       string | *""
	size:        int | *10 | string
	accessModes: [#AccessMode, ...#AccessMode] | #AccessMode | *"readWriteOnce"
}

#SecretBase: {
	labels:       [string]: string
	annotations:  [string]: string
}

#SecretOpaque: {
	#SecretBase
	type: "opaque"
	params?: [string]: _
	data: [string]:    string
}

#SecretTemplate: {
	#SecretBase
	type: "template"
	data: [string]: string
}

#SecretToken: {
	#SecretBase
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
	#SecretBase
	type: "basic"
	data: {
		username?: string
		password?: string
	}
}

#SecretGenerated: {
	#SecretBase
	type: "generated"
	params: {
		job:    string
		format: *"text" | "json"
	}
	data: {}
}

#Secret: *#SecretOpaque | #SecretBasicAuth | #SecretGenerated | #SecretTemplate | #SecretToken

#Router: {
	labels: [string]:      string
	annotations: [string]: string
	routes: [...#Route] | #RouteMap
}

#Route: {
	#RouteTarget
	path: =~#PathName
}

#RouteTarget: {
	pathType:          "exact" | *"prefix"
	targetServiceName: =~#DNSName
	targetPort?:       int
}

#RouteMap: [=~#PathName]: {
	=~#RouteTargetName | #RouteTarget
}

#RouteTargetName: "[a-z][-a-z0-9]*(:[0-9]+)?"

#PathName: "/.*"

#DNSName: "[a-z][-a-z0-9]*"

#Args: string | int | float | bool | [...string] | {...}

#App: {
	args: [string]: #Args
	profiles: [string]: [string]: #Args
	[=~"local[dD]ata"]: {...}
	containers: [=~#DNSName]: #Container
	jobs: [=~#DNSName]:       #Job
	images: [=~#DNSName]:     #Image
	volumes: [=~#DNSName]:    #Volume
	secrets: [=~#DNSName]:    #Secret
	routers: [=~#DNSName]:    #Router
	labels: [string]:         string
	annotations: [string]:    string
}
