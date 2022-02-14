package build

import (
	"github.com/ibuildthecloud/herd/schema/v1"
	"github.com/ibuildthecloud/herd/schema/v1/transform/normalize"
)

IN="in": v1.#App
_norm:   v1.#AppSpec & (normalize & {
	in: IN
}).out

out: {
	containers: {
		for k, v in _norm.containers {
			"\(k)": {
				image: string | *""
				image: v.image
				if v["build"] != _|_ {
                    build: v.build
				}
			}
		}
	}
	images: {
		for k, v in _norm.images {
			"\(k)": {
				image: string | *""
				image: v.image
				if v["build"] != _|_ {
                    build: v.build
				}
			}
		}
	}
}
