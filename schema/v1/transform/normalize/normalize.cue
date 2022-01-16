package normalize

import (
	"github.com/ibuildthecloud/herd/schema/v1"
	"path"
)

IN="in": v1.#App
out:     v1.#AppSpec & {
	containers: {
		for k, v in IN.containers {
			"\(k)": {
				image: v.image
				build: {
					dockerfile: path.Join([v.build, "Dockerfile"])
					context:    v.build
				} | v.build
			}
		}
	}
}
