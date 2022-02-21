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

	appImage, err = appImage.WithImageData(v1.ImageData{
		Images: map[string]v1.ContainerData{
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
		"root-path/sub/dir1/Dockerfile",
		"root-path/sub/dir2/Dockerfile",
		"root-path/sub/dir3/bockerfile",
	}, files)
}
