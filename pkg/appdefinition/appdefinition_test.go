package appdefinition

import (
	"os"
	"testing"

	"cuelang.org/go/cue/errors"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestContainerReferenceImage(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  ref: {
    image: images.full.image
  }
}

images: {
  full: {
    build: {
      context: "sub/dir2"	
      dockerfile: "sub/dir3/Dockerfile"
    }
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err := appImage.BuildSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Len(t, buildSpec.Containers, 1)
	assert.Equal(t, "", buildSpec.Containers["ref"].Image)
	assert.Nil(t, buildSpec.Containers["ref"].Build)

	assert.Len(t, buildSpec.Images, 1)
	assert.Equal(t, "", buildSpec.Images["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Images["full"].Build.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Images["full"].Build.Dockerfile)

	appImage, err = appImage.WithImageData(v1.ImagesData{
		Images: map[string]v1.ImageData{
			"full": {
				Image: "full-image",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err = appImage.BuildSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Len(t, buildSpec.Containers, 1)
	assert.Equal(t, "full-image", buildSpec.Containers["ref"].Image)
	assert.Nil(t, buildSpec.Containers["ref"].Build)

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}
	assert.Equal(t, "full-image", appSpec.Containers["ref"].Image)
	assert.Nil(t, appSpec.Containers["ref"].Build)
}

func TestAppImageBuildSpec(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  file: {
    build: "sub/dir1"
    sidecars: {
      left: {
        build: {}
      }
    }
  }
  none: {
    image: "done"
  }
  full: {
    build: {
      context: "sub/dir2"	
      dockerfile: "sub/dir3/Dockerfile"
      target: "other"
    }
  }
}

images: {
  file: {
    build: "sub/dir1"
  }
  none: {
    image: "done"
  }
  full: {
    build: {
      context: "sub/dir2"	
      dockerfile: "sub/dir3/Dockerfile"
      target: "other"
    }
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err := appImage.BuildSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Len(t, buildSpec.Containers, 3)
	assert.Equal(t, "", buildSpec.Containers["file"].Image)
	assert.Equal(t, "sub/dir1", buildSpec.Containers["file"].Build.Context)
	assert.Equal(t, "sub/dir1/Dockerfile", buildSpec.Containers["file"].Build.Dockerfile)
	assert.Equal(t, "", buildSpec.Containers["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Containers["full"].Build.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Containers["full"].Build.Dockerfile)
	assert.Equal(t, "other", buildSpec.Containers["full"].Build.Target)
	assert.Equal(t, "done", buildSpec.Containers["none"].Image)
	assert.Nil(t, buildSpec.Containers["none"].Build)

	assert.Equal(t, "Dockerfile", buildSpec.Containers["file"].Sidecars["left"].Build.Dockerfile)
	assert.Equal(t, ".", buildSpec.Containers["file"].Sidecars["left"].Build.Context)

	assert.Len(t, buildSpec.Images, 3)
	assert.Equal(t, "", buildSpec.Images["file"].Image)
	assert.Equal(t, "sub/dir1", buildSpec.Images["file"].Build.Context)
	assert.Equal(t, "sub/dir1/Dockerfile", buildSpec.Images["file"].Build.Dockerfile)
	assert.Equal(t, "", buildSpec.Images["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Images["full"].Build.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Images["full"].Build.Dockerfile)
	assert.Equal(t, "other", buildSpec.Images["full"].Build.Target)
	assert.Equal(t, "done", buildSpec.Images["none"].Image)
	assert.Nil(t, buildSpec.Images["none"].Build)
}

func TestWatchFiles(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  file: {
    build: "sub/dir1"
    sidecars: {
      left: {
        build: {
          dockerfile: "dockerfile.sidecar"
        }
      }
    }
  }
  none: {
    image: "done"
  }
  two: {
    build: {}
  }
  full: {
    build: {
	  dockerfile: "asdf/dockerfile"
    }
  }
}

images: {
  file: {
    build: "sub/dir2"
  }
  none: {
    image: "done"
  }
  two: {
    build: {}
  }
  full: {
    build: {
      dockerfile: "sub/dir3/bockerfile"
    }
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	files, err := appImage.WatchFiles("root-path")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{
		"root-path/Dockerfile",
		"root-path/asdf/dockerfile",
		"root-path/dockerfile.sidecar",
		"root-path/sub/dir1/Dockerfile",
		"root-path/sub/dir2/Dockerfile",
		"root-path/sub/dir3/bockerfile",
	}, files)
}

func TestEntrypoint(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
	entrypoint: "hi bye"
    image: "x"
  }
  a: {
	entrypoint: ["hi2","bye2"]
    image: "x"
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"hi", "bye"}, appSpec.Containers["s"].Entrypoint)
	assert.Equal(t, []string{"hi2", "bye2"}, appSpec.Containers["a"].Entrypoint)
}

func TestCommand(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
	command: "hi bye"
    image: "x"
  }
  a: {
	command: ["hi2","bye2"]
    image: "x"
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"hi", "bye"}, appSpec.Containers["s"].Command)
	assert.Equal(t, []string{"hi2", "bye2"}, appSpec.Containers["a"].Command)
}

func TestEnv(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    env: hi: "bye"
    image: ""
  }
  a: {
	env: ["hi2=bye2"]
    image: ""
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, []string{"hi=bye"}, appSpec.Containers["s"].Environment)
	assert.Equal(t, []string{"hi2=bye2"}, appSpec.Containers["a"].Environment)
}

func TestEnvironment(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    environment: hi: "bye"
    image: ""
  }
  a: {
	environment: ["hi2=bye2"]
    image: ""
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, []string{"hi=bye"}, appSpec.Containers["s"].Environment)
	assert.Equal(t, []string{"hi2=bye2"}, appSpec.Containers["a"].Environment)
}

func TestWorkdir(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    workdir: "something"
    image: ""
  }
  a: {
	workdir: "nothing"
    image: ""
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, "something", appSpec.Containers["s"].WorkingDir)
	assert.Equal(t, "nothing", appSpec.Containers["a"].WorkingDir)
}

func TestWorkingDir(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    workingDir: "something"
    image: ""
  }
  a: {
	workingDir: "nothing"
    image: ""
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, "something", appSpec.Containers["s"].WorkingDir)
	assert.Equal(t, "nothing", appSpec.Containers["a"].WorkingDir)
}

func TestCmd(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
	cmd: "hi bye"
    image: "x"
  }
  a: {
	cmd: ["hi2","bye2"]
    image: "x"
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"hi", "bye"}, appSpec.Containers["s"].Command)
	assert.Equal(t, []string{"hi2", "bye2"}, appSpec.Containers["a"].Command)
}

func TestInteractive(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
	interactive: true
    image: "x"
  }
  a: {
    image: "x"
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, appSpec.Containers["s"].Interactive)
	assert.False(t, appSpec.Containers["a"].Interactive)
}

func TestSidecar(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      init: true
	  image: "y"
	}
    image: "x"
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, "y", appSpec.Containers["s"].Sidecars["left"].Image)
	assert.True(t, appSpec.Containers["s"].Sidecars["left"].Init)
}

func TestPorts(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      image: "x"
      ports: [
        "80",
        "80:81",
        "80/http",
        "80:81/http",
	  ]
    }
	cmd: "hi bye"
    image: "x"
    ports: [
      "80",
      "80:81",
      "80/http",
      "80:81/http",
	]
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[1].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[3].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[1].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[3].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[3].Protocol, v1.ProtocolHTTP)
}

func TestFiles(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
	  image: "y"
	  files: {
	  	"/etc/something-sidecar": "bye"
	  	"/etc/something-sidecar-b": '\x00\x01bye'
	  }
	}
    image: "x"
	files: {
		"/etc/something": "hi"
	  	"/etc/something-b": '\x00\x01hi'
	}
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, "aGk=", appSpec.Containers["s"].Files["/etc/something"].Content)
	assert.Equal(t, "AAFoaQ==", appSpec.Containers["s"].Files["/etc/something-b"].Content)
	assert.Equal(t, "Ynll", appSpec.Containers["s"].Sidecars["left"].Files["/etc/something-sidecar"].Content)
	assert.Equal(t, "AAFieWU=", appSpec.Containers["s"].Sidecars["left"].Files["/etc/something-sidecar-b"].Content)
}

func TestVolumes(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      volumes: [
        "v1:/etc",
        {
          volume: "v2"
          mountPath: "/etc2"
          subPath: "a/b"
        }
      ]
	  image: "y"
	  files: {
	  	"/etc/something-sidecar": "bye"
	  }
	}
    image: "x"
    volumes: [
      "v1:/etc2",
      {
        volume: "v21"
        mountPath: "/etc3"
        subPath: "a/b2"
      }
    ]
	files: {
		"/etc/something": "hi"
	}
  }
}

volumes: {
  v1: {
    class: "aclass"
    size: 5
  }
  v2: {
    class: "bclass"
    size: 15
  }
  v21: {
    class: "cclass"
    size: 21
  }
  defaults: {
  }
}
`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, "aclass", appSpec.Volumes["v1"].Class)
	assert.Equal(t, int64(5), appSpec.Volumes["v1"].Size)
	assert.Equal(t, "bclass", appSpec.Volumes["v2"].Class)
	assert.Equal(t, int64(15), appSpec.Volumes["v2"].Size)
	assert.Equal(t, "cclass", appSpec.Volumes["v21"].Class)
	assert.Equal(t, int64(21), appSpec.Volumes["v21"].Size)
	assert.Equal(t, "", appSpec.Volumes["defaults"].Class)
	assert.Equal(t, int64(10), appSpec.Volumes["defaults"].Size)

	assert.Equal(t, "v1", appSpec.Containers["s"].Volumes[0].Volume)
	assert.Equal(t, "/etc2", appSpec.Containers["s"].Volumes[0].MountPath)
	assert.Equal(t, "v21", appSpec.Containers["s"].Volumes[1].Volume)
	assert.Equal(t, "/etc3", appSpec.Containers["s"].Volumes[1].MountPath)
	assert.Equal(t, "a/b2", appSpec.Containers["s"].Volumes[1].SubPath)

	assert.Equal(t, "v1", appSpec.Containers["s"].Sidecars["left"].Volumes[0].Volume)
	assert.Equal(t, "/etc", appSpec.Containers["s"].Sidecars["left"].Volumes[0].MountPath)
	assert.Equal(t, "v2", appSpec.Containers["s"].Sidecars["left"].Volumes[1].Volume)
	assert.Equal(t, "/etc2", appSpec.Containers["s"].Sidecars["left"].Volumes[1].MountPath)
	assert.Equal(t, "a/b", appSpec.Containers["s"].Sidecars["left"].Volumes[1].SubPath)

	assert.Equal(t, "aGk=", appSpec.Containers["s"].Files["/etc/something"].Content)
	assert.Equal(t, "Ynll", appSpec.Containers["s"].Sidecars["left"].Files["/etc/something-sidecar"].Content)
}
