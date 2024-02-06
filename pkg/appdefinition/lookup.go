package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"k8s.io/apimachinery/pkg/api/equality"
)

var semantic = equality.Semantic.Copy()

func init() {
	// Add custom equality functions
	z.Must(semantic.AddFunc(func(a, b v1.AcornBuild) bool {
		return a.OriginalImage == b.OriginalImage &&
			a.Context == b.Context &&
			a.Acornfile == b.Acornfile &&
			equality.Semantic.DeepEqual(a.BuildArgs.GetData(), b.BuildArgs.GetData())
	}))
}

func findImageInImageData(imageData v1.ImagesData, imageKey string) (string, bool) {
	var (
		parts         = strings.Split(imageKey, ".")
		containerName string
		sidecarName   string
	)
	if len(parts) > 2 {
		return "", false
	} else if len(parts) == 2 {
		containerName, sidecarName = parts[0], parts[1]
	} else {
		containerName = imageKey
	}
	if c, ok := imageData.Containers[containerName]; ok {
		if sidecarName != "" {
			s, ok := c.Sidecars[sidecarName]
			return s.Image, ok
		}
		return c.Image, true
	} else if c, ok := imageData.Functions[containerName]; ok {
		if sidecarName != "" {
			s, ok := c.Sidecars[sidecarName]
			return s.Image, ok
		}
		return c.Image, true
	} else if i, ok := imageData.Images[imageKey]; ok {
		return i.Image, true
	} else if j, ok := imageData.Jobs[containerName]; ok {
		if sidecarName != "" {
			s, ok := j.Sidecars[sidecarName]
			return s.Image, ok
		}
		return j.Image, true
	} else if a, ok := imageData.Acorns[imageKey]; ok {
		return a.Image, true
	}
	return "", false
}

func findContainerImage(imageData v1.ImagesData, image string, containerBuild *v1.Build) (string, bool) {
	if containerBuild == nil {
		if image != "" {
			for _, build := range imageData.Builds {
				if build.ContainerBuild != nil && build.ContainerBuild.Image == image && build.ImageKey != "" {
					return findImageInImageData(imageData, build.ImageKey)
				}
				if build.ImageBuild != nil && build.ImageBuild.Image == image && build.ImageKey != "" {
					return findImageInImageData(imageData, build.ImageKey)
				}
			}
		}
		return "", false
	}

	for _, build := range imageData.Builds {
		var testBuild *v1.Build
		if build.ContainerBuild != nil {
			testBuild = build.ContainerBuild.Build
		}
		if testBuild == nil && build.ImageBuild != nil {
			testBuild = build.ImageBuild.ContainerBuild
		}
		if testBuild == nil {
			continue
		}
		if !equality.Semantic.DeepEqual(*containerBuild, *testBuild) {
			continue
		}
		if build.ImageKey != "" {
			return findImageInImageData(imageData, build.ImageKey)
		}
		return "", false
	}

	return "", false
}

func isAutoUpgradePattern(image string) bool {
	return strings.ContainsAny(image, "*#")
}

func findAcornImage(imageData v1.ImagesData, autoUpgrade *bool, image string, acornBuild *v1.AcornBuild) (string, bool) {
	if isAutoUpgradePattern(image) || z.Dereference(autoUpgrade) {
		return image, image != ""
	}

	if acornBuild == nil {
		if image != "" {
			for _, build := range imageData.Builds {
				if build.ImageKey != "" && build.AcornBuild != nil && build.AcornBuild.Image == image && !build.AcornBuild.AutoUpgrade {
					return findImageInImageData(imageData, build.ImageKey)
				}
				if build.ImageKey != "" && build.ImageBuild != nil && build.ImageBuild.Image == image {
					return findImageInImageData(imageData, build.ImageKey)
				}
			}
		}
		return "", false
	}

	for _, build := range imageData.Builds {
		var (
			testBuild *v1.AcornBuild
			image     string
		)
		if build.AcornBuild != nil {
			testBuild = build.AcornBuild.Build
			image = build.AcornBuild.Image
		}
		if testBuild == nil && build.ImageBuild != nil {
			testBuild = build.ImageBuild.AcornBuild
		}
		if testBuild == nil {
			continue
		}
		if !semantic.DeepEqual(*acornBuild, *testBuild) {
			continue
		}
		if build.ImageKey != "" {
			return findImageInImageData(imageData, build.ImageKey)
		}
		return image, image != ""
	}
	return "", false
}

func GetImageReferenceForServiceName(svcName string, appSpec *v1.AppSpec, imageData v1.ImagesData) (result string, found bool) {
	var (
		parts         = strings.Split(svcName, ".")
		containerName string
		sidecarName   string
	)

	if len(parts) > 2 {
		return "", false
	} else if len(parts) == 2 {
		containerName, sidecarName = parts[0], parts[1]
	} else {
		containerName = svcName
	}

	if serviceDef, ok := appSpec.Services[svcName]; ok {
		return findAcornImage(imageData, serviceDef.AutoUpgrade, serviceDef.Image, serviceDef.Build)
	} else if acornDef, ok := appSpec.Acorns[svcName]; ok {
		return findAcornImage(imageData, acornDef.AutoUpgrade, acornDef.Image, acornDef.Build)
	} else if containerDef, ok := appSpec.Containers[containerName]; ok {
		if sidecarName != "" {
			containerDef, ok = containerDef.Sidecars[sidecarName]
			if !ok {
				return "", false
			}
		}
		result, ok := findContainerImage(imageData, containerDef.Image, containerDef.Build)
		// Only fall back to this check if there are no build records available, or this was a old build
		// that didn't record build with a context dir properly
		if !ok && notDirectReference(containerDef, imageData) {
			return findImageInImageData(imageData, svcName)
		}
		return result, ok
	} else if functionDef, ok := appSpec.Functions[containerName]; ok {
		if sidecarName != "" {
			functionDef, ok = functionDef.Sidecars[sidecarName]
			if !ok {
				return "", false
			}
		}
		result, ok := findContainerImage(imageData, functionDef.Image, functionDef.Build)
		// Only fall back to this check if there are no build records available, or this was a old build
		// that didn't record build with a context dir properly
		if !ok && notDirectReference(functionDef, imageData) {
			return findImageInImageData(imageData, svcName)
		}
		return result, ok
	} else if jobDef, ok := appSpec.Jobs[containerName]; ok {
		if sidecarName != "" {
			jobDef, ok = jobDef.Sidecars[sidecarName]
			if !ok {
				return "", false
			}
		}
		result, ok := findContainerImage(imageData, jobDef.Image, jobDef.Build)
		// Only fall back to this check if there are no build records available, or this was a old build
		// that didn't record build with a context dir properly
		if !ok && notDirectReference(jobDef, imageData) {
			return findImageInImageData(imageData, svcName)
		}
		return result, ok
	} else if imageDef, ok := appSpec.Images[svcName]; ok {
		if imageDef.Build != nil {
			return findContainerImage(imageData, "", imageDef.Build)
		} else if imageDef.AcornBuild != nil {
			return findContainerImage(imageData, "", imageDef.Build)
		}
		return findImageInImageData(imageData, svcName)
	}

	return "", false
}

func notDirectReference(con v1.Container, imageData v1.ImagesData) bool {
	// This is a direct image reference which should have been found earlier
	if len(imageData.Builds) > 0 && con.Image != "" && con.Build == nil {
		return false
	}
	return true
}
