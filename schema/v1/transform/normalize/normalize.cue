package normalize

import (
	"github.com/ibuildthecloud/herd/schema/v1"
	"path"
	"strings"
	"strconv"
	"encoding/base64"
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
		for k, v in IN.files {
			files: "\(k)": content: base64.Encode(null, v)
		}
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
		volumes: [ for v in IN.volumes {
			v | {
				_namePath: strings.SplitN(v, ":", 2)
				if len(_namePath) == 2 {
					volume:    _namePath[0]
					mountPath: _namePath[1]
				}
			}
		}]
		ports: [ for p in IN.ports {
			p | {
				_portProto: strings.SplitN(p, "/", 2)
				if len(_portProto) == 2 {
					protocol: _portProto[1]
				}
				_portPubPrivate: strings.SplitN(_portProto[0], ":", 2)
				port:            strconv.ParseInt(_portPubPrivate[0], 10, 32)
				if len(_portPubPrivate) == 2 {
					containerPort: strconv.ParseInt(_portPubPrivate[1], 10, 32)
				}
			}
		}]
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
	volumes: {
		for k, v in IN.volumes {
			"\(k)": v | {
				class: v.class
				size:  v.class
				accessMode: [v.class]
			}
		}
	}
}
