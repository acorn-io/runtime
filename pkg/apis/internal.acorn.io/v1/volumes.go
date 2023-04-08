package v1

import (
	"fmt"
	"strings"
)

func ParseVolumes(args []string, fromCLI bool) (result []VolumeBinding, _ error) {
	for _, arg := range args {
		arg, opts, _ := strings.Cut(arg, ",")
		existing, volName, ok := strings.Cut(arg, ":")
		if ok {
			if !fromCLI {
				return nil, fmt.Errorf("invalid volume configuration [%s], can not contain ':'", arg)
			}
		} else {
			volName = existing
			existing = ""
		}
		volName = strings.TrimSpace(volName)
		existing = strings.TrimSpace(existing)
		if volName == "" {
			return nil, fmt.Errorf("invalid volume name binding: [%s] must not have zero length value", arg)
		}
		volumeBinding := VolumeBinding{
			Volume: existing,
			Target: volName,
		}

		kvOpts := KVMap(opts, ",")
		if fromCLI {
			volumeBinding.Class = strings.TrimSpace(kvOpts["class"])
			q, err := ParseQuantity(kvOpts["size"])
			if err != nil {
				return nil, fmt.Errorf("parsing [%s]: %w", arg, err)
			}
			volumeBinding.Size = q
		} else if len(kvOpts) > 0 {
			return nil, fmt.Errorf("options [%s] are not supported in acorn volume binding definition", opts)
		}

		result = append(result, volumeBinding)
	}
	return
}
