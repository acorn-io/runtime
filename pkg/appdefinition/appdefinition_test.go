package appdefinition

import (
	"os"
	"testing"

	"cuelang.org/go/cue/errors"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestAppImageBuildSpec(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  file: {
    build: "sub/dir1"
    sidecars: {
      left: {
        build: {}
      }
      right: {
        image: "nginx"
        dirs: "/var/tmp": "./foo/bar"
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

	buildSpec, err := appImage.BuilderSpec()
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

	assert.Equal(t, "Dockerfile", buildSpec.Containers["file"].Sidecars["left"].Build.Dockerfile)
	assert.Equal(t, ".", buildSpec.Containers["file"].Sidecars["left"].Build.Context)

	assert.Equal(t, "nginx", buildSpec.Containers["file"].Sidecars["right"].Image)
	assert.Equal(t, "Dockerfile", buildSpec.Containers["file"].Sidecars["right"].Build.Dockerfile)
	assert.Equal(t, "nginx", buildSpec.Containers["file"].Sidecars["right"].Build.BaseImage)
	assert.Equal(t, ".", buildSpec.Containers["file"].Sidecars["right"].Build.Context)
	assert.Equal(t, "./foo/bar", buildSpec.Containers["file"].Sidecars["right"].Build.ContextDirs["/var/tmp"])

	assert.Len(t, buildSpec.Images, 3)
	assert.Equal(t, "", buildSpec.Images["file"].Image)
	assert.Equal(t, "sub/dir1", buildSpec.Images["file"].Build.Context)
	assert.Equal(t, "sub/dir1/Dockerfile", buildSpec.Images["file"].Build.Dockerfile)
	assert.Equal(t, "", buildSpec.Images["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Images["full"].Build.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Images["full"].Build.Dockerfile)
	assert.Equal(t, "other", buildSpec.Images["full"].Build.Target)
	assert.Equal(t, "done", buildSpec.Images["none"].Image)
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

	assert.Equal(t, []v1.EnvVar{{Name: "hi", Value: "bye"}}, appSpec.Containers["s"].Environment)
	assert.Equal(t, []v1.EnvVar{{Name: "hi2", Value: "bye2"}}, appSpec.Containers["a"].Environment)
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

	assert.Equal(t, []v1.EnvVar{{Name: "hi", Value: "bye"}}, appSpec.Containers["s"].Environment)
	assert.Equal(t, []v1.EnvVar{{Name: "hi2", Value: "bye2"}}, appSpec.Containers["a"].Environment)
}

func TestSecretDirs(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    dirs: "/var/tmp/foo": "secret://secname"
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

	assert.Equal(t, map[string]v1.VolumeMount{
		"/var/tmp/foo": {
			Secret: v1.VolumeSecretMount{
				Name: "secname",
			},
		},
	}, appSpec.Containers["s"].Dirs)
}

func TestSecretFiles(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    files: "/var/tmp/foo": "secret://secname/seckey"
    files: "/var/tmp/foo2": "nonsecret"
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

	assert.Equal(t, map[string]v1.File{
		"/var/tmp/foo": {
			Secret: v1.FileSecret{
				Name: "secname",
				Key:  "seckey",
			},
		},
		"/var/tmp/foo2": {
			Content: "bm9uc2VjcmV0",
		},
	}, appSpec.Containers["s"].Files)
}

func TestSecretEnv(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    environment: {
      hi: "bye"
      "secret://secname/seckey?optional=true": ""
      secretref: "secret://secname/seckey"
      secretrefembed: "secret://secname"
    }
    image: ""
  }
  a: {
	environment: [
      "hi=bye",
      "secret://secname/seckey?optional=true",
      "secretref=secret://secname/seckey",
      "secretrefembed=secret://secname",
    ]
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

	assert.Equal(t, []v1.EnvVar{
		{
			Secret: v1.EnvSecretVal{
				Name:     "secname",
				Key:      "seckey",
				Optional: &[]bool{true}[0],
			},
		},
		{
			Name:  "hi",
			Value: "bye",
		},
		{
			Name: "secretref",
			Secret: v1.EnvSecretVal{
				Name: "secname",
				Key:  "seckey",
			},
		},
		{
			Name: "secretrefembed",
			Secret: v1.EnvSecretVal{
				Name: "secname",
			},
		},
	}, appSpec.Containers["s"].Environment)
	assert.Equal(t, []v1.EnvVar{
		{
			Secret: v1.EnvSecretVal{
				Name:     "secname",
				Key:      "seckey",
				Optional: &[]bool{true}[0],
			},
		},
		{
			Name:  "hi",
			Value: "bye",
		},
		{
			Name: "secretref",
			Secret: v1.EnvSecretVal{
				Name: "secname",
				Key:  "seckey",
			},
		},
		{
			Name: "secretrefembed",
			Secret: v1.EnvSecretVal{
				Name: "secname",
			},
		},
	}, appSpec.Containers["a"].Environment)

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
		errors.Print(os.Stderr, err, nil)
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
  s1: {
	tty: true
    image: "x"
  }
  s2: {
	stdin: true
    image: "x"
  }
  s3: {
	tty: true
	stdin: true
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
	assert.True(t, appSpec.Containers["s1"].Interactive)
	assert.True(t, appSpec.Containers["s2"].Interactive)
	assert.True(t, appSpec.Containers["s3"].Interactive)
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

func TestPublish(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      image: "x"
      publish: [
        "80",
        "80:81",
        "80/http",
        "80:81/http",
	  ]
    }
    sidecars: right: {
      image: "x"
      publish: "80"
    }
    sidecars: right2: {
      image: "x"
      publish: 80
    }
	cmd: "hi bye"
    image: "x"
    publish: [
      80,
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
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	assert.Equal(t, true, appSpec.Containers["s"].Ports[0].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, true, appSpec.Containers["s"].Ports[1].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[1].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, true, appSpec.Containers["s"].Ports[2].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, true, appSpec.Containers["s"].Ports[3].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[3].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["left"].Ports[0].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["left"].Ports[1].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[1].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["left"].Ports[2].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["left"].Ports[3].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[3].ContainerPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["right"].Ports[0].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["right2"].Ports[0].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right2"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right2"].Ports[0].Protocol, v1.ProtocolTCP)
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
    sidecars: right: {
      image: "x"
      ports: "80"
    }
    sidecars: right2: {
      image: "x"
      ports: 80
    }
	cmd: "hi bye"
    image: "x"
    ports: [
      80,
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
		errors.Print(os.Stderr, err, nil)
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

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right"].Ports[0].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right2"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right2"].Ports[0].Protocol, v1.ProtocolTCP)
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

func TestImageBuildPermutations(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
	image: {
		image: "image-image"
        sidecars: side: image: "image-image-side"
	}
	build: {
        build: {}
        sidecars: side: build: {}
	}
	buildcontext: {
        build: {}
		dirs: "/var/tmp": "./foo/bar"
        sidecars: side: {
          build: {}
		  dirs: "/var/tmp": "./foo/bar"
        }
	}
	imagecontext: {
 		image: "imagecontext-image"
		dirs: "/var/tmp": "./foo/bar"
        sidecars: side: {
 		  image: "imagecontext-image-side",
		  dirs: "/var/tmp": "./foo/bar"
        }
	}
}
images: {
	build: {
      build: {}
    }
    image: {
      image: "images-image-image"
	}
}`))
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err := appImage.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, &v1.BuilderSpec{
		Jobs: map[string]v1.ContainerImageBuilderSpec{},
		Containers: map[string]v1.ContainerImageBuilderSpec{
			"image": {
				Image: "image-image",
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Image: "image-image-side",
					},
				},
			},
			"build": {
				Build: &v1.Build{
					Args:        map[string]string{},
					Context:     ".",
					Dockerfile:  "Dockerfile",
					ContextDirs: map[string]string{},
				},
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Build: &v1.Build{
							Args:        map[string]string{},
							Context:     ".",
							Dockerfile:  "Dockerfile",
							ContextDirs: map[string]string{},
						},
					},
				},
			},
			"buildcontext": {
				Build: &v1.Build{
					Args:       map[string]string{},
					Context:    ".",
					Dockerfile: "Dockerfile",
					ContextDirs: map[string]string{
						"/var/tmp": "./foo/bar",
					},
				},
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Build: &v1.Build{
							Args:       map[string]string{},
							Context:    ".",
							Dockerfile: "Dockerfile",
							ContextDirs: map[string]string{
								"/var/tmp": "./foo/bar",
							},
						},
					},
				},
			},
			"imagecontext": {
				Image: "imagecontext-image",
				Build: &v1.Build{
					Args:       map[string]string{},
					BaseImage:  "imagecontext-image",
					Context:    ".",
					Dockerfile: "Dockerfile",
					ContextDirs: map[string]string{
						"/var/tmp": "./foo/bar",
					},
				},
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Image: "imagecontext-image-side",
						Build: &v1.Build{
							Args:       map[string]string{},
							BaseImage:  "imagecontext-image-side",
							Context:    ".",
							Dockerfile: "Dockerfile",
							ContextDirs: map[string]string{
								"/var/tmp": "./foo/bar",
							},
						},
					},
				},
			},
		},
		Images: map[string]v1.ImageBuilderSpec{
			"build": {
				Build: &v1.Build{
					Args:        map[string]string{},
					Context:     ".",
					Dockerfile:  "Dockerfile",
					ContextDirs: map[string]string{},
				},
			},
			"image": {
				Image: "images-image-image",
			},
		},
	}, buildSpec)

	app := appImage.WithImageData(v1.ImagesData{
		Containers: map[string]v1.ContainerData{
			"image": {
				Image: "image-image",
				Sidecars: map[string]v1.ImageData{
					"side": {
						Image: "image-image-side",
					},
				},
			},
			"build": {
				Image: "build-image",
				Sidecars: map[string]v1.ImageData{
					"side": {
						Image: "build-image-side",
					},
				},
			},
			"buildcontext": {
				Image: "buildcontext-image",
				Sidecars: map[string]v1.ImageData{
					"side": {
						Image: "buildcontext-image-side",
					},
				},
			},
			"imagecontext": {
				Image: "imagecontext-image",
				Sidecars: map[string]v1.ImageData{
					"side": {
						Image: "imagecontext-image-side",
					},
				},
			},
		},
		Images: map[string]v1.ImageData{
			"build": {
				Image: "images-build-image",
			},
			"image": {
				Image: "images-image-image",
			},
		},
	})

	appSpec, err := app.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, appSpec.Containers, 4)
	assert.Len(t, appSpec.Images, 2)
	assert.Equal(t, "image-image", appSpec.Containers["image"].Image)
	assert.Equal(t, "image-image-side", appSpec.Containers["image"].Sidecars["side"].Image)
	assert.Equal(t, "build-image", appSpec.Containers["build"].Image)
	assert.Equal(t, "build-image-side", appSpec.Containers["build"].Sidecars["side"].Image)
	assert.Equal(t, "buildcontext-image", appSpec.Containers["buildcontext"].Image)
	assert.Equal(t, "buildcontext-image-side", appSpec.Containers["buildcontext"].Sidecars["side"].Image)
	assert.Equal(t, "imagecontext-image", appSpec.Containers["imagecontext"].Image)
	assert.Equal(t, "imagecontext-image-side", appSpec.Containers["imagecontext"].Sidecars["side"].Image)
}

func TestImageJSON(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
	simple: {
		build: "."
		dirs: "/test": "./files"
	}
}`))
	if err != nil {
		t.Fatal(err)
	}
	appImage = appImage.WithImageData(v1.ImagesData{
		Containers: map[string]v1.ContainerData{
			"simple": {
				Image: "hash",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hash", appSpec.Containers["simple"].Image)
}

func TestVolumes(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      directories: {
        "/var/short-vol": "short"
        "/var/short-implicit-vol": "short-implicit"
        "/var/not-ephemeral": "ephemeral"
        "/var/uri-vol": "volume://uri"
        "/var/uri-sub-vol": "volume://uri-sub?subPath=sub"
        "/var/uri-merge-vol": "volume://uri?class=uri-class&accessMode=readWriteMany&accessMode=readWriteOnce&size=7&size=5"
        "/var/anon-ephemeral-vol": ""
        "/var/anon-ephemeral2-vol": "ephemeral://"
        "/var/named-ephemeral-vol": "ephemeral://eph"
        "/var/named-context-vol": "./foo/bar"
      }
	  image: "y"
	}
    build: {
      context: "./foo/bar"
    }
    image: "built"
    dirs: {
      "/var/named-context-vol": "./sub"
    }
  }
}

volumes: {
  short: {
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

	appImage = appImage.WithImageData(v1.ImagesData{
		Containers: map[string]v1.ContainerData{
			"s": {
				Image: "image",
				Sidecars: map[string]v1.ImageData{
					"left": {
						Image: "image-side",
					},
				},
			},
		},
		Images: nil,
	})

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "aclass", appSpec.Volumes["short"].Class)
	assert.Equal(t, int64(5), appSpec.Volumes["short"].Size)
	assert.Equal(t, "bclass", appSpec.Volumes["v2"].Class)
	assert.Equal(t, int64(15), appSpec.Volumes["v2"].Size)
	assert.Equal(t, "cclass", appSpec.Volumes["v21"].Class)
	assert.Equal(t, int64(21), appSpec.Volumes["v21"].Size)
	assert.Equal(t, "", appSpec.Volumes["short-implicit"].Class)
	assert.Equal(t, int64(10), appSpec.Volumes["short-implicit"].Size)
	assert.Equal(t, int64(10), appSpec.Volumes["ephemeral"].Size)
	assert.Equal(t, "", appSpec.Volumes["defaults"].Class)
	assert.Equal(t, int64(10), appSpec.Volumes["defaults"].Size)
	assert.Equal(t, "uri-class", appSpec.Volumes["uri"].Class)
	assert.Equal(t, int64(5), appSpec.Volumes["uri"].Size)
	assert.Equal(t, []v1.AccessMode{"readWriteMany", "readWriteOnce"}, appSpec.Volumes["uri"].AccessModes)
	assert.Equal(t, int64(10), appSpec.Volumes["uri-sub"].Size)
	assert.Equal(t, []v1.AccessMode{"readWriteOnce"}, appSpec.Volumes["uri-sub"].AccessModes)
	assert.Equal(t, "ephemeral", appSpec.Volumes["left/var/anon-ephemeral-vol"].Class)
	assert.Equal(t, "ephemeral", appSpec.Volumes["left/var/anon-ephemeral2-vol"].Class)
	assert.Equal(t, "ephemeral", appSpec.Volumes["eph"].Class)
	assert.Len(t, typed.SortedKeys(appSpec.Volumes), 11)

	sidecar := appSpec.Containers["s"].Sidecars["left"]
	assert.Equal(t, "short", sidecar.Dirs["/var/short-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/short-vol"].SubPath)
	assert.Equal(t, "short-implicit", sidecar.Dirs["/var/short-implicit-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/short-implicit-vol"].SubPath)
	assert.Equal(t, "ephemeral", sidecar.Dirs["/var/not-ephemeral"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/not-ephemeral"].SubPath)
	assert.Equal(t, "uri", sidecar.Dirs["/var/uri-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/uri-vol"].SubPath)
	assert.Equal(t, "uri-sub", sidecar.Dirs["/var/uri-sub-vol"].Volume)
	assert.Equal(t, "sub", sidecar.Dirs["/var/uri-sub-vol"].SubPath)
	assert.Equal(t, "left/var/anon-ephemeral-vol", sidecar.Dirs["/var/anon-ephemeral-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/anon-ephemeral-vol"].SubPath)
	assert.Equal(t, "left/var/anon-ephemeral2-vol", sidecar.Dirs["/var/anon-ephemeral2-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/anon-ephemeral2-vol"].SubPath)
	assert.Equal(t, "eph", sidecar.Dirs["/var/named-ephemeral-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/named-ephemeral-vol"].SubPath)
	assert.Equal(t, "", sidecar.Dirs["/var/named-context-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/named-context-vol"].SubPath)
	assert.Equal(t, "./foo/bar", sidecar.Dirs["/var/named-context-vol"].ContextDir)
	assert.Equal(t, "./sub", appSpec.Containers["s"].Dirs["/var/named-context-vol"].ContextDir)
}

func TestSecrets(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    files: "/var/tmp/foo": "secret://file-implicit/key"
    files: "/var/tmp/foo-optional": "secret://file-implicit-opt/key?optional=true"
    dirs: "/var/tmp/": "secret://dirs-merge"
    dirs: "/var/tmp/opt": "secret://opt?optional=true"
    image: ""
  }
}
secrets: {
  explicit: {
    data: {
      username: "bardata"
    }
  }
  "dirs-merge": {
    optional: true
    type: "tls"
  }
  explicituser: {
    type: "basic"
    data: {
      username: "bardata"
      password: "barpass"
    }
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

	assert.Equal(t, v1.Secret{
		Type: "opaque",
		Data: map[string]string{
			"username": "bardata",
		},
	}, appSpec.Secrets["explicit"])
	assert.Equal(t, v1.Secret{
		Type: "basic",
		Data: map[string]string{
			"username": "bardata",
			"password": "barpass",
		},
	}, appSpec.Secrets["explicituser"])
	assert.Equal(t, v1.Secret{
		Type:     "opaque",
		Optional: &[]bool{true}[0],
		Data:     map[string]string{},
	}, appSpec.Secrets["file-implicit-opt"])
	assert.Equal(t, v1.Secret{
		Type: "opaque",
		Data: map[string]string{},
	}, appSpec.Secrets["file-implicit"])
	assert.Equal(t, v1.Secret{
		Type: "tls",
		Params: map[string]interface{}{
			"algorithm":    "ecdsa",
			"durationDays": 365.0,
			"usage":        "server",
			"sans":         []interface{}{},
			"organization": []interface{}{},
		},
		Optional: &[]bool{true}[0],
		Data:     map[string]string{},
	}, appSpec.Secrets["dirs-merge"])
	assert.Equal(t, v1.Secret{
		Type:     "opaque",
		Data:     map[string]string{},
		Optional: &[]bool{true}[0],
	}, appSpec.Secrets["opt"])

}

func TestImageDataOverride(t *testing.T) {
	herdCue := `
containers: db: image: "mariadb"
images: test: image: "another"
`
	image := &v1.ImagesData{
		Containers: map[string]v1.ContainerData{
			"db": {
				Image: "override-db",
			},
		},
		Images: map[string]v1.ImageData{
			"test": {
				Image: "override-image",
			},
		},
	}

	app, err := NewAppDefinition([]byte(herdCue))
	if err != nil {
		t.Fatal(err)
	}

	_, err = app.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	app = app.WithImageData(*image)
	appSpec, err := app.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "override-db", appSpec.Containers["db"].Image)
	assert.Equal(t, "override-image", appSpec.Images["test"].Image)
}

func TestJobs(t *testing.T) {
	herdCue := `
jobs: job1: image: "job1-image"
jobs: job2: image: "job2-image"
`
	app, err := NewAppDefinition([]byte(herdCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := app.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "job1-image", appSpec.Jobs["job1"].Image)
	assert.Equal(t, "job2-image", appSpec.Jobs["job2"].Image)
}

func TestNonUnique(t *testing.T) {
	herdCue := `
containers: foo: image: "test"
jobs: foo: image: "test"
`
	_, err := NewAppDefinition([]byte(herdCue))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "_keysMustBeUniqueAcrossTypes.foo: conflicting values \"jobs\" and \"container\"")
}

func TestFriendImageNameIsSet(t *testing.T) {
	herdCue := `
containers: foo: image: "test"
containers: foo: sidecars: side: image: "test"
containers: bar: {
  image: "test"
  dirs: "/var/lib": "./test"
}
jobs: job: image: "test"
images: image: image: "test"
`
	def, err := NewAppDefinition([]byte(herdCue))
	assert.Nil(t, err)
	appSpec, err := def.WithImageData(v1.ImagesData{
		Containers: map[string]v1.ContainerData{
			"foo": {
				Image: "foo-hash",
				Sidecars: map[string]v1.ImageData{
					"side": {
						Image: "side-hash",
					},
				},
			},
			"bar": {
				Image: "bar-hash",
			},
		},
		Jobs: map[string]v1.ContainerData{
			"job": {
				Image: "job-hash",
			},
		},
		Images: map[string]v1.ImageData{
			"image": {
				Image: "image-hash",
			},
		},
	}).AppSpec()
	assert.Nil(t, err)

	assert.Equal(t, "foo-hash", appSpec.Containers["foo"].Image)
	assert.Equal(t, "test", appSpec.Containers["foo"].Build.BaseImage)
	assert.Equal(t, "side-hash", appSpec.Containers["foo"].Sidecars["side"].Image)
	assert.Equal(t, "test", appSpec.Containers["foo"].Sidecars["side"].Build.BaseImage)
	assert.Equal(t, "bar-hash", appSpec.Containers["bar"].Image)
	assert.Equal(t, "test", appSpec.Containers["bar"].Build.BaseImage)
	assert.Equal(t, "job-hash", appSpec.Jobs["job"].Image)
	assert.Equal(t, "test", appSpec.Jobs["job"].Build.BaseImage)
	assert.Equal(t, "image-hash", appSpec.Images["image"].Image)
	assert.Equal(t, "test", appSpec.Images["image"].Build.BaseImage)
}

func TestBuildParameters(t *testing.T) {
	herdCue := `
params: build: {
  foo: string
}
containers: foo: build: args: one: params.build.foo
`
	def, err := NewAppDefinition([]byte(herdCue))
	if err != nil {
		t.Fatal(err)
	}

	def, err = def.WithBuildParams(map[string]interface{}{
		"foo": "two",
	})

	buildSpec, err := def.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "two", buildSpec.Containers["foo"].Build.Args["one"])
}
