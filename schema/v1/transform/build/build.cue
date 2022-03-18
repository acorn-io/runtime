package build

import (
	"github.com/ibuildthecloud/herd/schema/v1"
	"github.com/ibuildthecloud/herd/schema/v1/transform/normalize"
)

IN="in": v1.#App
out:     v1.#BuilderSpec
let _norm = v1.#AppSpec & (normalize & {
	in: {
		app: IN
	}
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

				for sk, sv in v.sidecars {
					sidecars: "\(sk)": {
						image: string | *""
						image: sv.image
						if sv["build"] != _|_ {
							build: sv.build
						}
					}
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
