package normalize

import (
	"github.com/ibuildthecloud/herd/schema/v1"
	"path"
	"strings"
)

#ToSidecar: {
	IN="in": _
	out: {
		{#ToContainer & {
			in: IN
		}}.out
		init: IN.init
	}
}

#ToContainer: {
	IN="in": _
	out: {
		image: IN.image
		if IN["build"] != _|_ {
			build: {
				dockerfile: path.Join([IN.build, "Dockerfile"])
				context:    IN.build
			} | IN.build
		}
		entrypoint: IN.entrypoint | strings.Split(IN.entrypoint, " ")
		if IN["command"] != _|_ {
			command: IN.command | strings.Split(IN.command, " ")
		}
		if IN["cmd"] != _|_ {
			command: IN.cmd | strings.Split(IN.cmd, " ")
		}
		if IN["env"] != _|_ {
			environment: IN.env | [ for k, v in IN.env {"\(k)=\(v)"}]
		}
		if IN["environment"] != _|_ {
			environment: IN.environment | [ for k, v in IN.environment {"\(k)=\(v)"}]
		}
		if IN["workdir"] != _|_ {
			workingDir: IN.workdir
		}
		if IN["workingDir"] != _|_ {
			workingDir: IN.workingDir
		}
		interactive: IN.interactive
	}
}

IN="in": v1.#App
out:     v1.#AppSpec & {
	containers: {
		for k, v in IN.containers {
			"\(k)": {
				{#ToContainer & {in: v}}.out
				for sk, sv in v.sidecars {
					sidecars: "\(sk)": {
						{#ToSidecar & {in: sv}}.out
					}
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
