package imagesource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/build"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/deployargs"
)

type ImageSource struct {
	Image     string
	File      string
	Args      []string
	ArgsFile  string
	Platforms []string
	// NoDefaultRegistry - if true, indicates that no container registry should be assumed for the Image.
	// This is used if the ImageSource is for an app with auto-upgrade enabled.
	NoDefaultRegistry bool

	// acornConfig is the path to the acorn config file.
	acornConfig string
}

func NewImageSource(acornConfig string, file, argsFile string, args, platforms []string, noDefaultReg bool) (result ImageSource) {
	result.acornConfig = acornConfig
	result.File = file
	result.Image, result.Args = splitImageAndArgs(args)
	result.ArgsFile = argsFile
	result.Platforms = platforms

	// If the image is a pattern, auto-upgrade is on, so assume no default registry
	_, isPattern := autoupgrade.AutoUpgradePattern(result.Image)
	result.NoDefaultRegistry = noDefaultReg || isPattern
	return
}

// isDirectory checks that the path from the provided directory
// point to a directory. If it does not point to a directory and it points at a file,
// it errors. Otherwise, the function returns false.
func isDirectory(cwd string) (bool, error) {
	if s, err := os.Stat(cwd); os.IsNotExist(err) {
		if strings.HasPrefix(cwd, ".") || strings.HasPrefix(cwd, "/") || strings.HasPrefix(cwd, "\\") {
			return false, fmt.Errorf("directory %s does not exist", cwd)
		}
		return false, nil
	} else if err != nil {
		return false, err
	} else if !s.IsDir() {
		return false, fmt.Errorf("%s is not a directory", cwd)
	}
	return true, nil
}

func (i ImageSource) IsImageSet() bool {
	return i.File != "" ||
		i.Image != ""
}

func (i ImageSource) GetAppDefinition(ctx context.Context, c client.Client) (*appdefinition.AppDefinition, map[string]any, []string, error) {
	image, file, err := i.ResolveImageAndFile()
	if err != nil {
		return nil, nil, nil, err
	}
	var (
		app        *appdefinition.AppDefinition
		sourceName string
	)
	if file == "" {
		sourceName = image
		imageDetails, err := c.ImageDetails(ctx, image, &client.ImageDetailsOptions{NoDefaultRegistry: i.NoDefaultRegistry})
		if err != nil {
			return nil, nil, nil, err
		}

		app, err = appdefinition.FromAppImage(&imageDetails.AppImage)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		sourceName = file
		app, err = build.ResolveAndParse(file)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	flags, err := deployargs.ToFlags(sourceName, i.ArgsFile, app)
	if err != nil {
		return nil, nil, nil, err
	}

	deployArgs, profiles, err := flags.Parse(i.Args)
	if err != nil {
		return nil, nil, nil, err
	}

	app = app.WithArgs(deployArgs, profiles)
	return app, deployArgs, profiles, nil
}

func (i ImageSource) WatchFiles(ctx context.Context, c client.Client) ([]string, error) {
	cwd, file, err := i.ResolveImageAndFile()
	if err != nil {
		return nil, err
	}
	if file == "" {
		// this is a reference to an image, not a build
		return nil, nil
	}

	app, _, _, err := i.GetAppDefinition(ctx, c)
	if err != nil {
		return []string{file}, err
	}

	files, err := app.WatchFiles(cwd)
	if err != nil {
		return []string{file}, err
	}

	return append([]string{file}, files...), nil
}

func (i ImageSource) ResolveImageAndFile() (string, string, error) {
	if !i.IsImageSet() {
		i.Image = "."
	}

	// at this point either i.Image or i.File is set, or both are set
	if i.Image == "" {
		// image is relative to i.File
		i.Image = filepath.Dir(i.File)
	} else if i.File == "" {
		// file is relative to i.Image if i.Image is a directory
		isDir, err := isDirectory(i.Image)
		if err != nil {
			return "", "", err
		}
		if isDir {
			if st, err := os.Stat(filepath.Join(i.Image, "Acorndir")); err == nil && st.IsDir() {
				i.File = filepath.Join(i.Image, "Acorndir")
			} else {
				i.File = filepath.Join(i.Image, "Acornfile")
			}
		}
	}
	return i.Image, i.File, nil
}

func (i ImageSource) WithImage(image string) ImageSource {
	i.Image = image
	return i
}

func (i ImageSource) GetImageAndDeployArgs(ctx context.Context, c client.Client) (string, map[string]any, []string, error) {
	var err error
	i.Image, i.File, err = i.ResolveImageAndFile()
	if err != nil {
		return "", nil, nil, err
	}

	// if file is set, then we must build to get the image, if it's not set, then
	// it must be an external image
	if i.File != "" {
		creds, err := GetCreds(i.acornConfig, c)
		if err != nil {
			return "", nil, nil, err
		}

		_, params, profiles, err := i.GetAppDefinition(ctx, c)
		if err != nil {
			return "", nil, nil, err
		}

		platforms, err := build.ParsePlatforms(i.Platforms)
		if err != nil {
			return "", nil, nil, err
		}

		image, err := c.AcornImageBuild(ctx, i.File, &client.AcornImageBuildOptions{
			Credentials: creds,
			Cwd:         i.Image,
			Args:        params,
			Profiles:    profiles,
			Platforms:   platforms,
		})
		if err != nil {
			return "", nil, nil, err
		}
		i.Image = image.ID
	}

	_, deployArgs, profiles, err := i.GetAppDefinition(ctx, c)
	return i.Image, deployArgs, profiles, err
}

func GetCreds(acornConfig string, c client.Client) (client.CredentialLookup, error) {
	cfg, err := config.ReadCLIConfig(acornConfig, false)
	if err != nil {
		return nil, err
	}

	creds, err := credentials.NewStore(cfg, c)
	if err != nil {
		return nil, err
	}
	return creds.Get, nil
}

func splitImageAndArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	if args[0] == "--" || args[0] == "" {
		return "", args[1:]
	}
	if strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}
