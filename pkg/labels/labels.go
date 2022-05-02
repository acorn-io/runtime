package labels

const (
	Prefix              = "acorn.io/"
	AcornAppNamespace   = Prefix + "app-namespace"
	AcornAppName        = Prefix + "app-name"
	AcornAcornName      = Prefix + "acorn-name"
	AcornAppUID         = Prefix + "app-uid"
	AcornVolumeName     = Prefix + "volume-name"
	AcornSecretName     = Prefix + "secret-name"
	AcornContainerName  = Prefix + "container-name"
	AcornJobName        = Prefix + "job-name"
	AcornAppImage       = Prefix + "app-image"
	AcornAppCuePath     = Prefix + "app-cue-path"
	AcornAppCuePathHash = Prefix + "app-cue-path-hash"
	AcornManaged        = Prefix + "managed"
	AcornContainerSpec  = Prefix + "container-spec"
	AcornImageMapping   = Prefix + "image-mapping"
	AcornPortNumber     = Prefix + "port-number"
	AcornHostnames      = Prefix + "hostnames"
	AcornAlias          = "alias." + Prefix
)
