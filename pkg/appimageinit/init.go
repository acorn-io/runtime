package appimageinit

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
)

func Print(cwd string, id string, output io.Writer) error {
	err := print(cwd, id, output)
	if err != nil {
		json.NewEncoder(output).Encode(map[string]interface{}{
			"error": err.Error(),
		})
	}
	return err
}

func print(cwd string, id string, output io.Writer) error {
	herdFile, err := ioutil.ReadFile(filepath.Join(cwd, appdefinition.HerdCueFile))
	if os.IsNotExist(err) {
		return fmt.Errorf("herd.cue is missing in app image: %s", id)
	} else if err != nil {
		return err
	}

	appImage := &v1.AppImage{
		ID:       id,
		Herdfile: string(herdFile),
	}

	imageReader, err := os.Open(filepath.Join(cwd, appdefinition.ImageDataFile))
	if err == nil {
		if err := json.NewDecoder(imageReader).Decode(&appImage.ImageData); err != nil {
			return fmt.Errorf("decoding %s in %s: %w", appdefinition.ImageDataFile, id, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("opening %s in %s: %w", appdefinition.ImageDataFile, id, err)
	}

	return json.NewEncoder(output).Encode(appImage)
}
