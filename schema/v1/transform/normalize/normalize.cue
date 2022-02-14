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
				if v["build"] != _|_ {
                    build: {
                        dockerfile: path.Join([v.build, "Dockerfile"])
                        context:    v.build
                    } | v.build
				}
			}
		}
	}
	images: {
		for k, v in IN.images {
			"\(k)": {
				image: v.image
				if v["build"] != _|_ {
                    build: {
                        dockerfile: path.Join([v.build, "Dockerfile"])
                        context:    v.build
                    } | v.build
				}
			}
		}
	}
}
