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
				build: v.build
			}
		}
	}
}
