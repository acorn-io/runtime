package labels

const (
	Prefix           = "herd-project.io/"
	HerdAppNamespace = Prefix + "app-namespace"
	HerdAppName      = Prefix + "app-name"
	// HerdAppPod is assigned to all pods that are associated to a herd app. This is used to identify
	// users pods from any system related pod in the namespace
	HerdAppPod         = Prefix + "pod"
	HerdContainerName  = Prefix + "container-name"
	HerdAppImage       = Prefix + "app-image"
	HerdAppCuePath     = Prefix + "app-cue-path"
	HerdAppCuePathHash = Prefix + "app-cue-path-hash"
)
