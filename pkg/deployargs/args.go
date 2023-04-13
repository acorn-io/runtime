package deployargs

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/flagparams"
	"golang.org/x/exp/maps"
)

func ToFlags(name string, appDef *appdefinition.AppDefinition) (*flagparams.Flags, error) {
	appSpec, err := appDef.AppSpec()
	if err != nil {
		return nil, err
	}

	params, err := appDef.Args()
	if err != nil {
		return nil, err
	}

	flags := flagparams.New(name, params)
	flags.Usage = Usage(appSpec)
	return flags, nil
}

func Usage(app *v1.AppSpec) func() {
	return func() {
		fmt.Println()
		if len(app.Volumes) == 0 {
			fmt.Println("Volumes:   <none>")
		} else {
			fmt.Print("Volumes:   ")
			fmt.Println(strings.Join(maps.Keys(app.Volumes), ", "))
		}

		if len(app.Secrets) == 0 {
			fmt.Println("Secrets:   <none>")
		} else {
			fmt.Print("Secrets:   ")
			fmt.Println(strings.Join(maps.Keys(app.Secrets), ", "))
		}

		if len(app.Containers) == 0 {
			fmt.Println("Containers: <none>")
		} else {
			fmt.Print("Containers: ")
			fmt.Println(strings.Join(maps.Keys(app.Containers), ", "))
		}

		var ports []string
		for containerName, container := range app.Containers {
			for _, port := range container.Ports {
				ports = append(ports, port.Complete().FormatString(containerName))
			}
			for _, sidecar := range container.Sidecars {
				for _, port := range sidecar.Ports {
					ports = append(ports, port.Complete().FormatString(containerName))
				}
			}
		}
		sort.Strings(ports)

		if len(ports) == 0 {
			fmt.Println("Ports:     <none>")
		} else {
			fmt.Print("Ports:     ")
			fmt.Println(strings.Join(ports, ", "))
		}

		fmt.Println()
	}
}
