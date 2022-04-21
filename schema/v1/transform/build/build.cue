package build

import (
	"github.com/acorn-io/acorn/schema/v1"
	"github.com/acorn-io/acorn/schema/v1/transform/normalize"
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
				if v["image"] != _|_ {
					image: v.image
				}
				if v["build"] != _|_ {
					build: v.build
				}

				for sk, sv in v.sidecars {
					sidecars: "\(sk)": {
						if sv["image"] != _|_ {
							image: sv.image
						}
						if sv["build"] != _|_ {
							build: sv.build
						}
					}
				}
			}
		}
	}
	jobs: {
		for k, v in _norm.jobs {
			"\(k)": {
				if v["image"] != _|_ {
					image: v.image
				}
				if v["build"] != _|_ {
					build: v.build
				}

				for sk, sv in v.sidecars {
					sidecars: "\(sk)": {
						if sv["image"] != _|_ {
							image: sv.image
						}
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
				if v["image"] != _|_ {
					image: v.image
				}
				if v["build"] != _|_ {
					build: v.build
				}
			}
		}
	}
}
