package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

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

func findContainerImage(imageData v1.ImagesData, containerBuild *v1.Build) (string, bool) {
	if containerBuild == nil {
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

func findAcornImage(imageData v1.ImagesData, image string, acornBuild *v1.AcornBuild) (string, bool) {
	if acornBuild == nil {
		for _, build := range imageData.Builds {
			if build.ImageKey == "" && build.AcornBuild != nil && build.AcornBuild.Image == image {
				return image, true
			}
			if build.ImageKey != "" && build.AcornBuild != nil && build.AcornBuild.Image == image && build.AcornBuild.Build == nil && !build.AcornBuild.AutoUpgrade {
				return findImageInImageData(imageData, build.ImageKey)
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
		if !equality.Semantic.DeepEqual(*acornBuild, *testBuild) {
			continue
		}
		if build.ImageKey != "" {
			return findImageInImageData(imageData, build.ImageKey)
		}
		return image, image != ""
	}
	return "", false
}

func GetImageReferenceForServiceName(svcName string, appSpec *v1.AppSpec, imageData v1.ImagesData) (string, bool) {
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

	image, ok := findImageInImageData(imageData, svcName)
	if ok {
		return image, true
	}

	if serviceDef, ok := appSpec.Services[svcName]; ok {
		return findAcornImage(imageData, serviceDef.Image, serviceDef.Build)
	} else if acornDef, ok := appSpec.Acorns[svcName]; ok {
		return findAcornImage(imageData, acornDef.Image, acornDef.Build)
	} else if containerDef, ok := appSpec.Containers[containerName]; ok {
		if sidecarName != "" {
			containerDef, ok = containerDef.Sidecars[sidecarName]
			if !ok {
				return "", false
			}
		}
		return findContainerImage(imageData, containerDef.Build)
	} else if jobDef, ok := appSpec.Jobs[containerName]; ok {
		if sidecarName != "" {
			jobDef, ok = jobDef.Sidecars[sidecarName]
			if !ok {
				return "", false
			}
		}
		return findContainerImage(imageData, jobDef.Build)
	} else if imageDef, ok := appSpec.Images[svcName]; ok {
		if imageDef.Build != nil {
			findContainerImage(imageData, imageDef.Build)
		} else if imageDef.AcornBuild != nil {
			findContainerImage(imageData, imageDef.Build)
		}
	}

	return "", false
}
