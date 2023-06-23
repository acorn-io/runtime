package appdefinition

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"cuelang.org/go/cue/errors"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRouters(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
routers: {
	slice: {
		routes: [
			{
				path: "/short"
				pathType: "exact"
				targetServiceName: "shorttarget"
				targetPort: 123
			},
			{
				path: "/longer"
				targetServiceName: "longertarget"
				targetPort: 1234
			},
		]
	}
	foo: {
		routes: {
			"/bar": "bartarget"
			"/aar": "aartarget"
			"/bar/exact": {
				pathType: "exact"
				targetServiceName: "barexacttarget"
				targetPort: 443
			}
			"/baazy": "foo:444"
		}
	}
}`))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v1.Router{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Routes: []v1.Route{
			{
				Path:              "/short",
				TargetServiceName: "shorttarget",
				TargetPort:        123,
				PathType:          v1.PathTypeExact,
			},
			{
				Path:              "/longer",
				TargetServiceName: "longertarget",
				TargetPort:        1234,
				PathType:          v1.PathTypePrefix,
			},
		},
	}, spec.Routers["slice"])

	assert.Equal(t, v1.Router{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Routes: []v1.Route{
			{
				Path:              "/bar/exact",
				TargetServiceName: "barexacttarget",
				TargetPort:        443,
				PathType:          v1.PathTypeExact,
			},
			{
				Path:              "/baazy",
				TargetServiceName: "foo",
				TargetPort:        444,
				PathType:          v1.PathTypePrefix,
			},
			{
				Path:              "/aar",
				TargetServiceName: "aartarget",
				TargetPort:        0,
				PathType:          v1.PathTypePrefix,
			},
			{
				Path:              "/bar",
				TargetServiceName: "bartarget",
				TargetPort:        0,
				PathType:          v1.PathTypePrefix,
			},
		},
	}, spec.Routers["foo"])
}

func TestParse5GLiteralVolume(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
volumes: {
  "data": {
    size: 5G
    accessModes: ["readWriteOnce"]
  }
  "data2": {
    size: 999999
    accessModes: ["readWriteOnce"]
  }
  "data3": {
    size: 1000000
    accessModes: ["readWriteOnce"]
  }
}`))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "5000000000", string(spec.Volumes["data"].Size))
	assert.Equal(t, "999999G", string(spec.Volumes["data2"].Size))
	assert.Equal(t, "1000000", string(spec.Volumes["data3"].Size))
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
      right: {
        image: "ghcr.io/acorn-io/images-mirror/nginx:latest"
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
  ifile: {
    acornBuild: "sub/dir1"
  }
  inone: {
    image: "done"
  }
  ifull: {
    containerBuild: {
      context: "sub/dir2"	
      dockerfile: "sub/dir3/Dockerfile"
      target: "other"
    }
  }
}

acorns: {
  afile: {
    build: "sub/dir1"
  }
  anone: {
    image: "done"
  }
  afull: {
    build: {
      context: "sub/dir2"
      acornfile: "sub/dir3/Acornfile"
      buildArgs: {
        key: "value"
        key2: {
          key3: "value3"
		}
	  }
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
	assert.Equal(t, filepath.Join("sub", "dir1", "Dockerfile"), buildSpec.Containers["file"].Build.Dockerfile)
	assert.Equal(t, "", buildSpec.Containers["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Containers["full"].Build.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Containers["full"].Build.Dockerfile)
	assert.Equal(t, "other", buildSpec.Containers["full"].Build.Target)
	assert.Equal(t, "done", buildSpec.Containers["none"].Image)

	assert.Equal(t, "Dockerfile", buildSpec.Containers["file"].Sidecars["left"].Build.Dockerfile)
	assert.Equal(t, ".", buildSpec.Containers["file"].Sidecars["left"].Build.Context)

	assert.Equal(t, "ghcr.io/acorn-io/images-mirror/nginx:latest", buildSpec.Containers["file"].Sidecars["right"].Image)
	assert.Equal(t, "Dockerfile", buildSpec.Containers["file"].Sidecars["right"].Build.Dockerfile)
	assert.Equal(t, "ghcr.io/acorn-io/images-mirror/nginx:latest", buildSpec.Containers["file"].Sidecars["right"].Build.BaseImage)
	assert.Equal(t, ".", buildSpec.Containers["file"].Sidecars["right"].Build.Context)
	assert.Equal(t, "./foo/bar", buildSpec.Containers["file"].Sidecars["right"].Build.ContextDirs["/var/tmp"])

	assert.Len(t, buildSpec.Images, 3)
	assert.Equal(t, "", buildSpec.Images["ifile"].Image)
	assert.Equal(t, "sub/dir1", buildSpec.Images["ifile"].AcornBuild.Context)
	assert.Equal(t, filepath.Join("sub", "dir1", "Acornfile"), buildSpec.Images["ifile"].AcornBuild.Acornfile)
	assert.Equal(t, "", buildSpec.Images["full"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Images["ifull"].ContainerBuild.Context)
	assert.Equal(t, "sub/dir3/Dockerfile", buildSpec.Images["ifull"].ContainerBuild.Dockerfile)
	assert.Equal(t, "other", buildSpec.Images["ifull"].ContainerBuild.Target)
	assert.Equal(t, "done", buildSpec.Images["inone"].Image)

	assert.Len(t, buildSpec.Acorns, 3)
	assert.Equal(t, "", buildSpec.Acorns["afile"].Image)
	assert.Equal(t, "sub/dir1", buildSpec.Acorns["afile"].Build.Context)
	assert.Equal(t, filepath.Join("sub", "dir1", "Acornfile"), buildSpec.Acorns["afile"].Build.Acornfile)
	assert.Equal(t, "", buildSpec.Acorns["afull"].Image)
	assert.Equal(t, "sub/dir2", buildSpec.Acorns["afull"].Build.Context)
	assert.Equal(t, "sub/dir3/Acornfile", buildSpec.Acorns["afull"].Build.Acornfile)
	assert.Equal(t, "value", buildSpec.Acorns["afull"].Build.BuildArgs["key"])
	assert.Equal(t, map[string]any{"key3": "value3"}, buildSpec.Acorns["afull"].Build.BuildArgs["key2"])
	assert.Equal(t, "done", buildSpec.Acorns["anone"].Image)
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
  ifile: {
    acornBuild: "sub/dir2"
  }
  inone: {
    image: "done"
  }
  itwo: {
    acornBuild: {}
  }
  ifull: {
    containerBuild: {
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
		filepath.Join("root-path", ".dockerignore"),
		filepath.Join("root-path", "Acornfile"),
		filepath.Join("root-path", "Dockerfile"),
		filepath.Join("root-path", "asdf", ".dockerignore"),
		filepath.Join("root-path", "asdf", "dockerfile"),
		filepath.Join("root-path", "dockerfile.sidecar"),
		filepath.Join("root-path", "sub", "dir1", ".dockerignore"),
		filepath.Join("root-path", "sub", "dir1", "Dockerfile"),
		filepath.Join("root-path", "sub", "dir2", "Acornfile"),
		filepath.Join("root-path", "sub", "dir3", "bockerfile"),
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

	assert.Equal(t, v1.CommandSlice{"hi", "bye"}, appSpec.Containers["s"].Entrypoint)
	assert.Equal(t, v1.CommandSlice{"hi2", "bye2"}, appSpec.Containers["a"].Entrypoint)
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

	assert.Equal(t, v1.CommandSlice{"hi", "bye"}, appSpec.Containers["s"].Command)
	assert.Equal(t, v1.CommandSlice{"hi2", "bye2"}, appSpec.Containers["a"].Command)
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

	assert.Equal(t, v1.EnvVars{{Name: "hi", Value: "bye"}}, appSpec.Containers["s"].Environment)
	assert.Equal(t, v1.EnvVars{{Name: "hi2", Value: "bye2"}}, appSpec.Containers["a"].Environment)
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

	assert.Equal(t, v1.EnvVars{{Name: "hi", Value: "bye"}}, appSpec.Containers["s"].Environment)
	assert.Equal(t, v1.EnvVars{{Name: "hi2", Value: "bye2"}}, appSpec.Containers["a"].Environment)
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
				Name:     "secname",
				OnChange: v1.ChangeTypeRedeploy,
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

	assert.Equal(t, v1.Files{
		"/var/tmp/foo": {
			Mode: "",
			Secret: v1.SecretReference{
				Name:     "secname",
				Key:      "seckey",
				OnChange: v1.ChangeTypeRedeploy,
			},
		},
		"/var/tmp/foo2": {
			Mode:    "0644",
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
      "secret://secname/seckey?onchange=no-action": ""
      secretref: "secret://secname/seckey"
      secretrefembed: "secret://secname"
    }
    image: ""
  }
  a: {
	environment: [
      "hi=bye",
      "secret://secname/seckey?onchange=redeploy",
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

	assert.Equal(t, v1.EnvVars{
		{
			Secret: v1.SecretReference{
				Name:     "secname",
				Key:      "seckey",
				OnChange: v1.ChangeTypeNoAction,
			},
		},
		{
			Name:  "hi",
			Value: "bye",
		},
		{
			Name: "secretref",
			Secret: v1.SecretReference{
				Name:     "secname",
				Key:      "seckey",
				OnChange: v1.ChangeTypeRedeploy,
			},
		},
		{
			Name: "secretrefembed",
			Secret: v1.SecretReference{
				Name:     "secname",
				OnChange: v1.ChangeTypeRedeploy,
			},
		},
	}, appSpec.Containers["s"].Environment)
	assert.Equal(t, v1.EnvVars{
		{
			Secret: v1.SecretReference{
				Name:     "secname",
				Key:      "seckey",
				OnChange: v1.ChangeTypeRedeploy,
			},
		},
		{
			Name:  "hi",
			Value: "bye",
		},
		{
			Name: "secretref",
			Secret: v1.SecretReference{
				Name:     "secname",
				Key:      "seckey",
				OnChange: v1.ChangeTypeRedeploy,
			},
		},
		{
			Name: "secretrefembed",
			Secret: v1.SecretReference{
				Name:     "secname",
				OnChange: v1.ChangeTypeRedeploy,
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

	assert.Equal(t, v1.CommandSlice{"hi", "bye"}, appSpec.Containers["s"].Command)
	assert.Equal(t, v1.CommandSlice{"hi2", "bye2"}, appSpec.Containers["a"].Command)
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

func TestExpose(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
     image: "x"
     ports: expose: [
      "80",
      "80:81/tcp",
      "80/http",
      "80:81/http",
	 ]
    }
    sidecars: left2: {
     image: "x"
     ports: publish: 80
     ports: expose: "81/http"
    }
    sidecars: right: {
     image: "x"
     ports: "80"
    }
    sidecars: right2: {
     image: "x"
     ports: 80
    }
    sidecars: right3: {
     image: "x"
     ports: [{targetPort: 12}, 80]
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
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[1].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[1].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(0), appSpec.Containers["s"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[3].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["left"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[1].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["left"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[3].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, false, appSpec.Containers["s"].Sidecars["left2"].Ports[0].Publish)
	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["left2"].Ports[0].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left2"].Ports[0].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left2"].Ports[0].Protocol, v1.ProtocolHTTP)
	assert.Equal(t, true, appSpec.Containers["s"].Sidecars["left2"].Ports[1].Publish)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left2"].Ports[1].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left2"].Ports[1].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left2"].Ports[1].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["right"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right2"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right2"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(12), appSpec.Containers["s"].Sidecars["right3"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right3"].Ports[0].Protocol, v1.Protocol(""))
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right3"].Ports[1].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right3"].Ports[1].Protocol, v1.Protocol(""))
}

func TestPorts(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
      image: "x"
      ports: [
        "80",
        "80:81/tcp",
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
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[0].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[1].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[1].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(0), appSpec.Containers["s"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[2].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Ports[3].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["left"].Ports[0].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[0].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[1].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[1].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[1].Protocol, v1.ProtocolTCP)

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["left"].Ports[2].Port)
	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[2].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[2].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["left"].Ports[3].Port)
	assert.Equal(t, int32(81), appSpec.Containers["s"].Sidecars["left"].Ports[3].TargetPort)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["left"].Ports[3].Protocol, v1.ProtocolHTTP)

	assert.Equal(t, int32(0), appSpec.Containers["s"].Sidecars["right"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right"].Ports[0].Protocol, v1.Protocol(""))

	assert.Equal(t, int32(80), appSpec.Containers["s"].Sidecars["right2"].Ports[0].Port)
	assert.Equal(t, appSpec.Containers["s"].Sidecars["right2"].Ports[0].Protocol, v1.Protocol(""))
}

func TestFiles(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: {
  s: {
    sidecars: left: {
	  image: "y"
	  files: {
	  	"/etc/something-sidecar": "bye"
		"/bin/secret.sh": "secret://foo/bar?mode=123"
	  }
	}
    image: "x"
	files: {
		"/etc/something": "hi"
	  	"/exec.sh": "blah"
	  	"/bin/exec": "blah"
	  	"/sbin/exec": "blah"
		"/full": {
			content: "blah"
			mode: "0127"
		}
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
	assert.Equal(t, "0644", appSpec.Containers["s"].Files["/etc/something"].Mode)
	assert.Equal(t, "0755", appSpec.Containers["s"].Files["/exec.sh"].Mode)
	assert.Equal(t, "0755", appSpec.Containers["s"].Files["/bin/exec"].Mode)
	assert.Equal(t, "0755", appSpec.Containers["s"].Files["/sbin/exec"].Mode)
	assert.Equal(t, "blah", appSpec.Containers["s"].Files["/full"].Content)
	assert.Equal(t, "0127", appSpec.Containers["s"].Files["/full"].Mode)
	assert.Equal(t, "Ynll", appSpec.Containers["s"].Sidecars["left"].Files["/etc/something-sidecar"].Content)
	assert.Equal(t, "123", appSpec.Containers["s"].Sidecars["left"].Files["/bin/secret.sh"].Mode)
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
	ibuild: {
      acornBuild: {}
    }
    iimage: {
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
		Services: map[string]v1.AcornBuilderSpec{},
		Jobs:     map[string]v1.ContainerImageBuilderSpec{},
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
					BuildArgs:  map[string]string{},
					Context:    ".",
					Dockerfile: "Dockerfile",
				},
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Build: &v1.Build{
							BuildArgs:  map[string]string{},
							Context:    ".",
							Dockerfile: "Dockerfile",
						},
					},
				},
			},
			"buildcontext": {
				Build: &v1.Build{
					BuildArgs:  map[string]string{},
					Context:    ".",
					Dockerfile: "Dockerfile",
					ContextDirs: map[string]string{
						"/var/tmp": "./foo/bar",
					},
				},
				Sidecars: map[string]v1.ContainerImageBuilderSpec{
					"side": {
						Build: &v1.Build{
							BuildArgs:  map[string]string{},
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
			"ibuild": {
				AcornBuild: &v1.AcornBuild{
					BuildArgs: v1.GenericMap{},
					Context:   ".",
					Acornfile: "Acornfile",
				},
			},
			"iimage": {
				Image: "images-image-image",
			},
		},
		Acorns: map[string]v1.AcornBuilderSpec{},
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
			"ibuild": {
				Image: "images-build-image",
			},
			"iimage": {
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
        "/var/uri-merge-vol": "volume://uri?class=uri-class&accessMode=readWriteMany&accessMode=readWriteOnce&size=70&size=50"
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

	appImage, _, err = appImage.WithArgs(map[string]any{"foo": "bar"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	toQuantity := func(i int64) v1.Quantity {
		q, _ := v1.ParseQuantity(strconv.Itoa(int(i)))
		return q
	}
	assert.Equal(t, "aclass", appSpec.Volumes["short"].Class)
	assert.Equal(t, toQuantity(5), appSpec.Volumes["short"].Size)
	assert.Equal(t, "bclass", appSpec.Volumes["v2"].Class)
	assert.Equal(t, toQuantity(15), appSpec.Volumes["v2"].Size)
	assert.Equal(t, "cclass", appSpec.Volumes["v21"].Class)
	assert.Equal(t, toQuantity(21), appSpec.Volumes["v21"].Size)
	assert.Equal(t, "", appSpec.Volumes["short-implicit"].Class)
	assert.Equal(t, v1.Quantity(""), appSpec.Volumes["short-implicit"].Size)
	assert.Equal(t, v1.Quantity(""), appSpec.Volumes["ephemeral"].Size)
	assert.Equal(t, "", appSpec.Volumes["defaults"].Class)
	assert.Equal(t, v1.Quantity(""), appSpec.Volumes["defaults"].Size)
	assert.Nil(t, appSpec.Volumes["defaults"].AccessModes)
	// class not supported
	assert.Equal(t, "", appSpec.Volumes["uri"].Class)
	assert.Equal(t, toQuantity(70), appSpec.Volumes["uri"].Size)
	assert.Equal(t, v1.AccessModes{"readWriteMany", "readWriteOnce"}, appSpec.Volumes["uri"].AccessModes)
	assert.Equal(t, toQuantity(10), appSpec.Volumes["uri-sub"].Size)
	assert.Nil(t, appSpec.Volumes["uri-sub"].AccessModes)
	assert.Equal(t, "ephemeral", appSpec.Volumes["s/left/var/anon-ephemeral-vol"].Class)
	assert.Equal(t, "ephemeral", appSpec.Volumes["s/left/var/anon-ephemeral2-vol"].Class)
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
	assert.Equal(t, filepath.Join("s", "left", "var", "anon-ephemeral-vol"), sidecar.Dirs["/var/anon-ephemeral-vol"].Volume)
	assert.Equal(t, "", sidecar.Dirs["/var/anon-ephemeral-vol"].SubPath)
	assert.Equal(t, filepath.Join("s", "left", "var", "anon-ephemeral2-vol"), sidecar.Dirs["/var/anon-ephemeral2-vol"].Volume)
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
    files: "/var/tmp/foo-optional": "secret://file-implicit-opt/key?onchange=redeploy"
    dirs: "/var/tmp/": "secret://dirs-merge"
    dirs: "/var/tmp/opt": "secret://opt?onchange=no-action"
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
    type: "opaque"
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
		Type:        "opaque",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Data: map[string]string{
			"username": "bardata",
		},
	}, appSpec.Secrets["explicit"])
	assert.Equal(t, v1.Secret{
		Type:        "basic",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Data: map[string]string{
			"username": "bardata",
			"password": "barpass",
		},
	}, appSpec.Secrets["explicituser"])
	assert.Equal(t, v1.Secret{
		Type: "opaque",
	}, appSpec.Secrets["file-implicit-opt"])
	assert.Equal(t, v1.Secret{
		Type: "opaque",
	}, appSpec.Secrets["file-implicit"])
	assert.Equal(t, v1.Secret{
		Type:        "opaque",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Data:        map[string]string{},
	}, appSpec.Secrets["dirs-merge"])
	assert.Equal(t, v1.Secret{
		Type: "opaque",
	}, appSpec.Secrets["opt"])
}

func TestImageDataOverride(t *testing.T) {
	acornCue := `
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

	app, err := NewAppDefinition([]byte(acornCue))
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
	acornCue := `
jobs: job1: image: "job1-image"
jobs: job2: image: "job2-image"
`
	app, err := NewAppDefinition([]byte(acornCue))
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
	acornCue := `
containers: foo: image: "test"
jobs: foo: image: "test"
`
	_, err := NewAppDefinition([]byte(acornCue))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "duplicate name [foo] used by [container] and [job]")
}

func TestFriendImageNameIsSet(t *testing.T) {
	acornCue := `
containers: foo: image: "test"
containers: foo: sidecars: side: image: "test"
containers: bar: {
  image: "test"
  dirs: "/var/lib": "./test"
}
jobs: job: image: "test"
images: image: image: "test"
`
	def, err := NewAppDefinition([]byte(acornCue))
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
	assert.Equal(t, "bar-hash", appSpec.Containers["bar"].Build.BaseImage)
	assert.Equal(t, "job-hash", appSpec.Jobs["job"].Image)
	assert.Equal(t, "test", appSpec.Jobs["job"].Build.BaseImage)
	assert.Equal(t, "image-hash", appSpec.Images["image"].Image)
	assert.Equal(t, "test", appSpec.Images["image"].Build.BaseImage)
}

func TestScale(t *testing.T) {
	acornCue := `
containers: nil: {}
containers: one: scale: 1
containers: zero: scale: 0
`
	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, (*int32)(nil), appSpec.Containers["nil"].Scale)
	assert.Equal(t, int32(1), *appSpec.Containers["one"].Scale)
	assert.Equal(t, int32(0), *appSpec.Containers["zero"].Scale)
}

func TestBuildProfileParameters(t *testing.T) {
	acornCue := `
args: {
  foo: "three"
}
profiles: one: foo: string | *"one"
profiles: two: foo: string | *"two"
containers: foo: build: buildArgs: one: args.foo
`
	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = def.WithArgs(map[string]any{}, []string{"one", "two", "three"})
	assert.Equal(t, "failed to find profile three", err.Error())

	def, _, err = def.WithArgs(map[string]any{}, []string{"one", "two", "three?"})
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err := def.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "one", buildSpec.Containers["foo"].Build.BuildArgs["one"])

	def, _, err = def.WithArgs(map[string]any{}, []string{"two", "one"})
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err = def.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "two", buildSpec.Containers["foo"].Build.BuildArgs["one"])
}

func TestBuildParameters(t *testing.T) {
	acornCue := `
args: {
  foo: "bad"
}
containers: foo: build: buildArgs: one: args.foo
`
	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	def, _, err = def.WithArgs(map[string]any{
		"foo": "two",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	buildSpec, err := def.BuilderSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "two", buildSpec.Containers["foo"].Build.BuildArgs["one"])
}

func TestAcornFirstForm(t *testing.T) {
	acornCue := `
acorns: first: {
	labels: [
		{
			resourceType: "container"
			resourceName: "containerName"
			key: "keyname"
			value: "value"
		},
		{
			resourceType: "acorn"
			resourceName: "containerName2"
			key: "keyname2"
			value: "value2"
		}
	]
	annotations: [
		{
			resourceType: "container"
			resourceName: "containerName"
			key: "keyname"
			value: "value"
		},
		{
			resourceType: "acorn"
			resourceName: "containerName2"
			key: "keyname2"
			value: "value2"
		}
	]
	image: "acornimage"
	build: "./dir"
	publish: 80
	volumes: "b"
	secrets: "a:b"
	links: "b:c"
	autoUpgrade: true
	autoUpgradeInterval: "20s"
	notifyUpgrade: true
	mem: 1Gi
	env: [
		"a=b",
		"c=d"
	]
	deployArgs: foo: 12
	profiles: ["abc", "def"]
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	acorn := appSpec.Acorns["first"]

	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "container",
		ResourceName: "containerName",
		Key:          "keyname",
		Value:        "value",
	}, acorn.Labels[0])
	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "acorn",
		ResourceName: "containerName2",
		Key:          "keyname2",
		Value:        "value2",
	}, acorn.Labels[1])
	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "container",
		ResourceName: "containerName",
		Key:          "keyname",
		Value:        "value",
	}, acorn.Annotations[0])
	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "acorn",
		ResourceName: "containerName2",
		Key:          "keyname2",
		Value:        "value2",
	}, acorn.Annotations[1])
	assert.Equal(t, "acornimage", acorn.Image)
	assert.Equal(t, &v1.AcornBuild{
		Context:   "./dir",
		Acornfile: "dir/Acornfile",
	}, acorn.Build)
	assert.Equal(t, v1.VolumeBinding{
		Volume: "",
		Target: "b",
	}, acorn.Volumes[0])
	assert.Equal(t, v1.SecretBinding{
		Secret: "a",
		Target: "b",
	}, acorn.Secrets[0])
	assert.Equal(t, v1.ServiceBinding{
		Service: "b",
		Target:  "c",
	}, acorn.Links[0])
	assert.True(t, *acorn.AutoUpgrade)
	assert.True(t, *acorn.NotifyUpgrade)
	assert.Equal(t, "20s", acorn.AutoUpgradeInterval)
	assert.Equal(t, v1.MemoryMap{
		"": &[]int64{1 << 30}[0],
	}, acorn.Memory)
	assert.Equal(t, v1.NameValue{
		Name:  "a",
		Value: "b",
	}, acorn.Environment[0])
	assert.Equal(t, v1.NameValue{
		Name:  "c",
		Value: "d",
	}, acorn.Environment[1])
	assert.Equal(t, v1.GenericMap{
		"foo": int64(12),
	}, acorn.DeployArgs)
	assert.Equal(t, []string{"abc", "def"}, acorn.Profiles)
}

func TestAcornSecondForm(t *testing.T) {
	acornCue := `
acorns: first: {
	labels: {
		"containers:containername:abc.io/keyname": "value",
	}
	annotations: {
		"acorn:containername2:abc.io/keyname2": "value2",
	}
	build: {
		context: "abc"
		acornfile: "other/Acornfile"
		buildArgs: a: "b"
	}
	publish: "example.com:1234/http"
	volumes: [{
		target: "def"
	}]
	secrets: [{
		secret: "abc"
		target: "def"
	}]
	links: [{
		service: "abc"
		target: "def"
	}]
	mem: {
		"foo": 1Gi
	}
	env: {
		"X": "v"
	}
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	acorn := appSpec.Acorns["first"]

	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "container",
		ResourceName: "containername",
		Key:          "abc.io/keyname",
		Value:        "value",
	}, acorn.Labels[0])
	assert.Equal(t, v1.ScopedLabel{
		ResourceType: "acorn",
		ResourceName: "containername2",
		Key:          "abc.io/keyname2",
		Value:        "value2",
	}, acorn.Annotations[0])
	assert.Equal(t, &v1.AcornBuild{
		Context:   "abc",
		Acornfile: "other/Acornfile",
		BuildArgs: v1.GenericMap{
			"a": "b",
		},
	}, acorn.Build)
	assert.Equal(t, v1.PortBinding{
		Hostname:   "example.com",
		TargetPort: 1234,
		Protocol:   v1.ProtocolHTTP,
	}, acorn.Publish[0])
	assert.Equal(t, v1.VolumeBinding{
		Volume: "",
		Target: "def",
	}, acorn.Volumes[0])
	assert.Equal(t, v1.SecretBinding{
		Secret: "abc",
		Target: "def",
	}, acorn.Secrets[0])
	assert.Equal(t, v1.ServiceBinding{
		Service: "abc",
		Target:  "def",
	}, acorn.Links[0])
	assert.Equal(t, v1.MemoryMap{
		"foo": &[]int64{1 << 30}[0],
	}, acorn.Memory)
	assert.Equal(t, v1.NameValues{
		{
			Name:  "X",
			Value: "v",
		},
	}, acorn.Environment)
}

func TestAcornThirdForm(t *testing.T) {
	acornCue := `
acorns: first: {
	publish: [{
		hostname: "example.com"
		targetPort: 80
		targetServiceName: "asdf"
	}, 80, "example.com:123"]
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	acorn := appSpec.Acorns["first"]

	assert.Equal(t, v1.PortBinding{
		Hostname:          "example.com",
		Port:              80,
		TargetPort:        80,
		TargetServiceName: "asdf",
	}, acorn.Publish[0])
	assert.Equal(t, v1.PortBinding{
		TargetPort: 80,
	}, acorn.Publish[1])
	assert.Equal(t, v1.PortBinding{
		Hostname:   "example.com",
		TargetPort: 123,
		Protocol:   v1.ProtocolHTTP,
	}, acorn.Publish[2])
}

func TestService(t *testing.T) {
	acornCue := `
services: first: {
	default: true
	external: "externalname"
	address: "some.address"
	ports: "80"
	container: "foo"
	secrets: ["sfoo", "sbar"]
	data: foo: hi: "bye"
}

services: acorn: {
	default: true
	labels: foo: "bar"
	image: "foo"
	build: "./dir"
	secrets: "foo:bar"
	env: foo: "bar"
	autoUpgrade: true
	autoUpgradeInterval:   "5m"
	notifyUpgrade:         true
	mem:      44G
	env: foo: "bar"
	serviceArgs: key: "value"
}

services: job: {
	default: true
	labels: foo: "bar"
	generated: job: "foo"
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	svc := appSpec.Services["first"]

	assert.Equal(t, true, svc.Default)
	assert.Equal(t, "externalname", svc.External)
	assert.Equal(t, "some.address", svc.Address)
	assert.Equal(t, v1.PortDef{
		TargetPort: 80,
	}, svc.Ports[0])
	assert.Equal(t, "foo", svc.Container)
	assert.Equal(t, v1.SecretBindings{
		{
			Secret: "sfoo",
			Target: "sfoo",
		},
		{
			Secret: "sbar",
			Target: "sbar",
		},
	}, svc.Secrets)
	assert.Equal(t, v1.GenericMap{
		"foo": map[string]any{
			"hi": "bye",
		},
	}, svc.Data)

	acorn := appSpec.Services["acorn"]

	assert.Equal(t, true, acorn.Default)
	assert.Equal(t, "foo", acorn.Image)
	assert.Equal(t, "./dir", acorn.Build.Context)
	assert.Equal(t, "foo", acorn.Labels[0].Key)
	assert.Equal(t, "bar", acorn.Labels[0].Value)
	assert.Equal(t, "foo", acorn.Image)
	assert.Equal(t, "./dir", acorn.Build.Context)
	assert.Equal(t, v1.SecretBindings{
		{
			Secret: "foo",
			Target: "bar",
		},
	}, acorn.Secrets)
	assert.Equal(t, true, *acorn.AutoUpgrade)
	assert.Equal(t, "5m", acorn.AutoUpgradeInterval)
	assert.Equal(t, true, *acorn.NotifyUpgrade)
	assert.Equal(t, int64(44000000000), *acorn.Memory[""])
	assert.Equal(t, "foo", acorn.Environment[0].Name)
	assert.Equal(t, "bar", acorn.Environment[0].Value)
	assert.Equal(t, v1.GenericMap{
		"key": "value",
	}, acorn.ServiceArgs)

	job := appSpec.Services["job"]
	assert.Equal(t, true, job.Default)
	assert.Equal(t, "foo", job.Generated.Job)
	assert.Equal(t, "foo", job.Labels[0].Key)
	assert.Equal(t, "bar", job.Labels[0].Value)
}

func TestCronJob(t *testing.T) {
	acornCue := `
jobs: foo: {
  image: "image"
  schedule: "daily"
}`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "daily", appSpec.Jobs["foo"].Schedule)
}

func TestInvalidPublishHostname(t *testing.T) {
	acornCue := `
containers: foo: {
  ports: "foo2:80"
  image: "image"
}
containers: foo2: image: "image"
`

	_, err := NewAppDefinition([]byte(acornCue))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[foo2] is not a valid hostname to publish, missing \".\"")
}

func TestLink(t *testing.T) {
	acornCue := `
acorns: one: links: ["two:three", "one"]
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "two", appSpec.Acorns["one"].Links[0].Service)
	assert.Equal(t, "three", appSpec.Acorns["one"].Links[0].Target)
	assert.Equal(t, "one", appSpec.Acorns["one"].Links[1].Service)
	assert.Equal(t, "one", appSpec.Acorns["one"].Links[1].Target)
}

func TestAlias(t *testing.T) {
	acornCue := `
containers: foo: {
  ports: "foo2.local:80"
  image: "image"
}
containers: foo3: {
  ports: "foo4.local:80"
  image: "image"
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo2.local", appSpec.Containers["foo"].Ports[0].Hostname)
	assert.Equal(t, "foo4.local", appSpec.Containers["foo3"].Ports[0].Hostname)
}

func TestProbes(t *testing.T) {
	acornCue := `
containers: tcp: {
	probe: "tcp://localhost:1234"
}
containers: http: {
	probe: "http://localhost:1234"
}
containers: https: {
	probe: "https://localhost:1234"
}
containers: cmd: {
	probe: "/usr/bin/true"
}
containers: spec: {
	probes: [{
		type: "startup"
		exec: command: ["/usr/bin/false"]
		initialDelaySeconds: 1
		timeoutSeconds:      2
		periodSeconds :      3
		successThreshold:    4
		failureThreshold:    5
	}]
}
containers: map: {
	probe: liveness: "/usr/bin/true"
	probe: startup: {
		exec: command: ["/usr/bin/false"]
		initialDelaySeconds: 1
		timeoutSeconds:      2
		periodSeconds :      3
		successThreshold:    4
		failureThreshold:    5
	}
	probe: ready: "/usr/bin/true"
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v1.ProbeType("readiness"), appSpec.Containers["tcp"].Probes[0].Type)
	assert.Equal(t, "tcp://localhost:1234", appSpec.Containers["tcp"].Probes[0].TCP.URL)

	assert.Equal(t, v1.ProbeType("readiness"), appSpec.Containers["http"].Probes[0].Type)
	assert.Equal(t, "http://localhost:1234", appSpec.Containers["http"].Probes[0].HTTP.URL)

	assert.Equal(t, v1.ProbeType("readiness"), appSpec.Containers["https"].Probes[0].Type)
	assert.Equal(t, "https://localhost:1234", appSpec.Containers["https"].Probes[0].HTTP.URL)

	assert.Equal(t, v1.ProbeType("readiness"), appSpec.Containers["cmd"].Probes[0].Type)
	assert.Equal(t, []string{"/usr/bin/true"}, appSpec.Containers["cmd"].Probes[0].Exec.Command)

	assert.Equal(t, v1.ProbeType("liveness"), appSpec.Containers["map"].Probes[0].Type)
	assert.Equal(t, []string{"/usr/bin/true"}, appSpec.Containers["map"].Probes[0].Exec.Command)
	assert.Equal(t, v1.ProbeType("readiness"), appSpec.Containers["map"].Probes[1].Type)
	assert.Equal(t, []string{"/usr/bin/true"}, appSpec.Containers["map"].Probes[1].Exec.Command)
	assert.Equal(t, v1.ProbeType("startup"), appSpec.Containers["map"].Probes[2].Type)
	assert.Equal(t, []string{"/usr/bin/false"}, appSpec.Containers["map"].Probes[2].Exec.Command)
	assert.Equal(t, int32(1), appSpec.Containers["map"].Probes[2].InitialDelaySeconds)
	assert.Equal(t, int32(2), appSpec.Containers["map"].Probes[2].TimeoutSeconds)
	assert.Equal(t, int32(3), appSpec.Containers["map"].Probes[2].PeriodSeconds)
	assert.Equal(t, int32(4), appSpec.Containers["map"].Probes[2].SuccessThreshold)
	assert.Equal(t, int32(5), appSpec.Containers["map"].Probes[2].FailureThreshold)

	assert.Equal(t, v1.ProbeType("startup"), appSpec.Containers["spec"].Probes[0].Type)
	assert.Equal(t, []string{"/usr/bin/false"}, appSpec.Containers["spec"].Probes[0].Exec.Command)
	assert.Equal(t, int32(1), appSpec.Containers["spec"].Probes[0].InitialDelaySeconds)
	assert.Equal(t, int32(2), appSpec.Containers["spec"].Probes[0].TimeoutSeconds)
	assert.Equal(t, int32(3), appSpec.Containers["spec"].Probes[0].PeriodSeconds)
	assert.Equal(t, int32(4), appSpec.Containers["spec"].Probes[0].SuccessThreshold)
	assert.Equal(t, int32(5), appSpec.Containers["spec"].Probes[0].FailureThreshold)
}

func TestDepsSingle(t *testing.T) {
	acornCue := `
containers: default: {
	probe: "tcp://localhost:1234"
	dependsOn: "foo"
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, appSpec.Containers["default"].Dependencies, 1)
	assert.Equal(t, "foo", appSpec.Containers["default"].Dependencies[0].TargetName)
}

func TestDepsMultiple(t *testing.T) {
	acornCue := `
containers: default: {
	probe: "tcp://localhost:1234"
	depends_on: ["foo", "bar"]
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, appSpec.Containers["default"].Dependencies, 2)
	assert.Equal(t, "foo", appSpec.Containers["default"].Dependencies[0].TargetName)
	assert.Equal(t, "bar", appSpec.Containers["default"].Dependencies[1].TargetName)
}

func TestDontFailIfProfileDoesntHaveBuildOrDeploy(t *testing.T) {
	acornCue := `
profiles: foo: {}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = def.WithArgs(nil, []string{"foo"})
	assert.Nil(t, err)

	_, _, err = def.WithArgs(nil, []string{"missing"})
	assert.Equal(t, "failed to find profile missing", err.Error())
}

func TestPermissions(t *testing.T) {
	acornCue := `
localData: permissions: {
  rules: [
    {
      verbs: ["verb"]
      apiGroups: ["groups"]
      resources: ["resources"]
      resourceNames: ["names"]
      nonResourceURLs: ["foo"]
      scopes: ["foo"]
    }
  ]
}

containers: cont: {
  permissions: localData.permissions
  sidecars: side: permissions: localData.permissions
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "verb", appSpec.Containers["cont"].Permissions.Rules[0].Verbs[0])
	assert.Equal(t, "groups", appSpec.Containers["cont"].Permissions.Rules[0].APIGroups[0])
	assert.Equal(t, "resources", appSpec.Containers["cont"].Permissions.Rules[0].Resources[0])
	assert.Equal(t, "names", appSpec.Containers["cont"].Permissions.Rules[0].ResourceNames[0])
	assert.Equal(t, "foo", appSpec.Containers["cont"].Permissions.Rules[0].NonResourceURLs[0])
	assert.Equal(t, "foo", appSpec.Containers["cont"].Permissions.Rules[0].Scopes[0])

	assert.Equal(t, "verb", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].Verbs[0])
	assert.Equal(t, "groups", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].APIGroups[0])
	assert.Equal(t, "resources", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].Resources[0])
	assert.Equal(t, "names", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].ResourceNames[0])
	assert.Equal(t, "foo", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].NonResourceURLs[0])
	assert.Equal(t, "foo", appSpec.Containers["cont"].Sidecars["side"].Permissions.Rules[0].Scopes[0])
}

func TestNoImport(t *testing.T) {
	acornCue := `
import (
  "list"
)
localData: { r: list.Range(1,2,3) }
`

	_, err := NewAppDefinition([]byte(acornCue))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "import keyword is not")
}

func TestNoPackage(t *testing.T) {
	acornCue := `
package foo
localData: {}
`

	_, err := NewAppDefinition([]byte(acornCue))
	assert.NotNil(t, err)
}

func TestCustomFunc(t *testing.T) {
	acornCue := `
containers: foo: image: localData.data
localData: {
    data: echo("hi")
	data: "hi"
	echo: {
		args: [_]
		out: args[0]
	}
}
`

	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hi", appSpec.Containers["foo"].Image)
}

func TestStdMissing(t *testing.T) {
	data := `
let foo = std.toyaml({})
`

	_, err := NewAppDefinition([]byte(data))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid reference to std.toyaml, closest matches [toYAML fromYAML trim]")
}

func TestArgsTopCondition(t *testing.T) {
	data := `
let foo = true
if foo {
	args: {
		// test comment
		test: "adsf",
	}
}
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	args, err := appDef.Args()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "test comment", args.Params[0].Description)
}

func TestArgsRejectConditions(t *testing.T) {
	data := `
args: {
	if foo {
		test: "adsf",
	}
}
`
	_, err := NewAppDefinition([]byte(data))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "comprehension (if) should not be used inside the args and profiles fields")
}

func getVals(t *testing.T, appDef *AppDefinition) map[string]any {
	t.Helper()

	data := map[string]any{}
	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	b, err := base64.StdEncoding.DecodeString(appSpec.Containers["default"].Files["a"].Content)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(b, &data)
	if err != nil {
		t.Fatalf("%s: %v", string(b), err)
	}

	return data
}

func TestNoConditionInProfile(t *testing.T) {
	data := `
profiles: default: {
	if true {
		a: "b"
	}
}
`
	_, err := NewAppDefinition([]byte(data))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "comprehension (if) should not be used inside the args and profiles fields")
}

func TestNoConditionInProfileTop(t *testing.T) {
	data := `
profiles: {
	if true {
		default: {
			a: "b"
		}
	}
}
`
	_, err := NewAppDefinition([]byte(data))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "comprehension (if) should not be used inside the args and profiles fields")
}

func TestProfileDefaultValues(t *testing.T) {
	data := `
containers: default: files: a: std.toJSON(args)
// include some non-alpha characters
profiles: "foo - bar": {
	a: "b"
}
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	defaultAppDef, args, err := appDef.WithArgs(nil, []string{"foo - bar"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, map[string]any{"a": "b"}, args)
	assert.Equal(t, map[string]any{"a": "b", "dev": false}, getVals(t, defaultAppDef))

	appDef, args, err = appDef.WithArgs(map[string]any{"a": "c", "c": "d"}, []string{"foo - bar"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, map[string]any{"a": "c", "c": "d"}, args)
	assert.Equal(t, map[string]any{"a": "c", "c": "d", "dev": false}, getVals(t, appDef))
}

func TestArgsDefaulting(t *testing.T) {
	data := `
args: {
	s: "s"
	i: 4
	f: 6.0
	b: true
	bn: false
	e: "x" | "y" | "z"
	a: ["val"]
	o: {}
}
containers: default: files: "a": std.toJSON(args)
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, map[string]any{
		"dev": false,
		"s":   "s",
		"i":   4.0,
		"f":   6.0,
		"b":   true,
		"bn":  false,
		"e":   "x",
		"a":   []any{"val"},
		"o":   map[string]any{},
	}, getVals(t, appDef))

	newValues := map[string]any{
		"dev": true,
		"s":   "s2",
		"i":   5.0,
		"f":   5.1,
		"b":   false,
		"bn":  true,
		"a":   []any{"1", "2", "3"},
		"o": map[string]any{
			"x": "y",
		},
	}

	appDef, _, err = appDef.WithArgs(newValues, nil)
	if err != nil {
		t.Fatal(err)
	}

	newValues["e"] = "x"
	assert.Equal(t, newValues, getVals(t, appDef))
}

func TestErrorFriendly(t *testing.T) {
	data := `
foo: int
foo: bar
bar: baz
baz: "h"
`
	_, err := NewAppDefinition([]byte(data))
	assert.NotNil(t, err)
	assert.Equal(t, `foo: conflicting values int and "h" (mismatched types int and string):
    Acornfile:2:6
    Acornfile:3:6
    Acornfile:4:6
    Acornfile:5:6
`, err.Error())
}

func TestEnvValFromArg(t *testing.T) {
	data := `
args: a: "foo"
containers: a: env: a: args.a
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "a", appSpec.Containers["a"].Environment[0].Name)
	assert.Equal(t, "foo", appSpec.Containers["a"].Environment[0].Value)
}

func TestEmptyEnvVal(t *testing.T) {
	data := `
containers: a: env: a: ""
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "a", appSpec.Containers["a"].Environment[0].Name)
	assert.Equal(t, "", appSpec.Containers["a"].Environment[0].Value)
}

func TestEmptyAcornEnv(t *testing.T) {
	data := `
acorns: slice: env: ["a=b", "c=d"]
acorns: m: env: a: "b"
acorns: m: env: c: "d"
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, appSpec.Acorns["slice"].Environment, 2)
	assert.Equal(t, "a", appSpec.Acorns["slice"].Environment[0].Name)
	assert.Equal(t, "b", appSpec.Acorns["slice"].Environment[0].Value)
	assert.Equal(t, "c", appSpec.Acorns["slice"].Environment[1].Name)
	assert.Equal(t, "d", appSpec.Acorns["slice"].Environment[1].Value)

	assert.Len(t, appSpec.Acorns["m"].Environment, 2)
	assert.Equal(t, "a", appSpec.Acorns["m"].Environment[0].Name)
	assert.Equal(t, "b", appSpec.Acorns["m"].Environment[0].Value)
	assert.Equal(t, "c", appSpec.Acorns["m"].Environment[1].Name)
	assert.Equal(t, "d", appSpec.Acorns["m"].Environment[1].Value)
}

func TestTemplateSecretCustomNames(t *testing.T) {
	data := `
containers: test: {
	image: "foo"
	env: foo: "secret://template/foo"
}
secrets: template: {
    type: "template"
	data: foo: "yep"
}
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "yep", appSpec.Secrets["template"].Data["foo"])
}

func TestDefaultContextDir(t *testing.T) {
	data := `
containers: test: build: "./foo"
containers: test2: build: context: "./foo"
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo/Dockerfile", appSpec.Containers["test"].Build.Dockerfile)
	assert.Equal(t, "foo/Dockerfile", appSpec.Containers["test2"].Build.Dockerfile)
}

func TestShortPermissions(t *testing.T) {
	data := `
containers: test: {
	image: "foo"
	permissions: {
		rules: [
			"pods.api.group",
			"read secrets"
		]
	}
}
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "pods", appSpec.Containers["test"].Permissions.Rules[0].Resources[0])
	assert.Equal(t, "api.group", appSpec.Containers["test"].Permissions.Rules[0].APIGroups[0])
	assert.Equal(t, v1.DefaultVerbs, appSpec.Containers["test"].Permissions.Rules[0].Verbs)
	assert.Equal(t, "secrets", appSpec.Containers["test"].Permissions.Rules[1].Resources[0])
	assert.Equal(t, "api.acorn.io", appSpec.Containers["test"].Permissions.Rules[1].APIGroups[0])
	assert.Equal(t, []string{"get", "list", "watch"}, appSpec.Containers["test"].Permissions.Rules[1].Verbs)
}

func TestMultipleEphemeralBug(t *testing.T) {
	data := `
containers: test: {
	image: "foo"
	dirs: "/foo": "ephemeral://blah"
	dirs: "/foo2": "ephemeral://blah"
	dirs: "/foo3": "ephemeral://blah"
}
`
	appDef, err := NewAppDefinition([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "blah", appSpec.Containers["test"].Dirs["/foo"].Volume)
	assert.Equal(t, "blah", appSpec.Containers["test"].Dirs["/foo2"].Volume)
	assert.Equal(t, "blah", appSpec.Containers["test"].Dirs["/foo3"].Volume)
}

func TestDisableProbes(t *testing.T) {
	appImage, err := NewAppDefinition([]byte(`
containers: map: probes: {}
containers: array: probes: []
containers: default: image: "foo"
`))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := appImage.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v1.Probes{}, spec.Containers["array"].Probes)
	assert.Equal(t, v1.Probes{}, spec.Containers["map"].Probes)
	assert.Equal(t, v1.Probes(nil), spec.Containers["default"].Probes)
}

func TestNestedScopedLabels(t *testing.T) {
	// labels and annotations on a acorn are both unmarshalled into a ScopedLabels struct, which is just a slice
	// Similar to ports, in the Acornfile you can define them using an object syntax or short-form string syntax.
	// We're testing that they get unmarshalled properly
	appImage, err := NewAppDefinition([]byte(`acorns: {
	s: {
		labels: [
			{
				resourceType: "",
				resourceName: "",
				key: "globalkey",
				value: "x0"
			},
			{
				resourceType: "containers",
				resourceName: "",
				key: "conskey",
				value: "x1"
			},
			{
				resourceType: "containers",
				resourceName: "confoo",
				key: "fookey",
				value: "x2"
			},
			{
				resourceType: "",
				resourceName: "objbar",
				key: "acorn.io/barkey",
				value: "x3"
			}
		]
		annotations: {
			globalkey: "v0"
			"containers:conskey": "v1"
			"containers:confoo:acorn.io/fookey": "v2"
			"conbar:barkey": "v3"
		}
		image: ""
	}}`))
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := appImage.AppSpec()
	if err != nil {
		errors.Print(os.Stderr, err, nil)
		t.Fatal(err)
	}

	// Asserting that labels defined using the struct/object form are properly unmarshalled
	assert.Equal(t, v1.ScopedLabel{ResourceType: "", ResourceName: "", Key: "globalkey", Value: "x0"}, appSpec.Acorns["s"].Labels[0])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "container", ResourceName: "", Key: "conskey", Value: "x1"}, appSpec.Acorns["s"].Labels[1])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "container", ResourceName: "confoo", Key: "fookey", Value: "x2"}, appSpec.Acorns["s"].Labels[2])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "", ResourceName: "objbar", Key: "acorn.io/barkey", Value: "x3"}, appSpec.Acorns["s"].Labels[3])

	// Asserting that annotations defined using the shorthand string form are properly unmarshalled
	assert.Equal(t, v1.ScopedLabel{ResourceType: "", ResourceName: "", Key: "globalkey", Value: "v0"}, appSpec.Acorns["s"].Annotations[0])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "", ResourceName: "conbar", Key: "barkey", Value: "v3"}, appSpec.Acorns["s"].Annotations[1])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "container", ResourceName: "", Key: "conskey", Value: "v1"}, appSpec.Acorns["s"].Annotations[2])
	assert.Equal(t, v1.ScopedLabel{ResourceType: "container", ResourceName: "confoo", Key: "acorn.io/fookey", Value: "v2"}, appSpec.Acorns["s"].Annotations[3])

	// Expect error because resourceType isn't one of the whitelisted types
	_, err = NewAppDefinition([]byte(`acorns: {
	s: {
    	labels: [
			{
				resourceType: "asdf",
				resourceName: "",
				key: "key",
				value: "x0"
			},
		]
		image: ""
	}}`))
	assert.Error(t, err)

	// Expect error because label key's resourceName section isn't a simple dnsName
	_, err = NewAppDefinition([]byte(`acorns: {
	s: {
    	labels: {
			"containers:acorn.io/foo:key": "val"
		}
		image: ""
	}}`))
	assert.Error(t, err)

	// Expect error because resourceType isn't a simple dns name
	_, err = NewAppDefinition([]byte(`acorns: {
	s: {
    	labels: [
			{
				resourceType: "acorn.io"
				key: "asdf"
				value: "asdf"
			}
		]
		image: ""
	}}`))
	assert.Error(t, err)
}
