package normalize

import (
	"github.com/acorn-io/acorn/schema/v1"
	"strings"
)

#ToProbeSpec: {
	IN="in": {
		def:  string | v1.#ProbeSpec
		type: string
	}
	out: v1.#ProbeSpec
	out: {
		if (IN.def & string) != _|_ {
			if strings.HasPrefix(IN.def, "http://") {
				http: url: IN.def
			}
			if strings.HasPrefix(IN.def, "https://") {
				http: url: IN.def
			}
			if strings.HasPrefix(IN.def, "tcp://") {
				tcp: url: IN.def
			}
			if !strings.HasPrefix(IN.def, "http://") &&
				!strings.HasPrefix(IN.def, "https://") &&
				!strings.HasPrefix(IN.def, "tcp://") {
				exec: command: strings.Split(IN.def, " ")
			}
		}
		if (IN.def & v1.#ProbeSpec) != _|_ {
			IN.def
		}
		type: IN.type
	}
}

#ToProbeSpecs: {
	IN="in": v1.#Probes
	out: [...v1.#ProbeSpec]
	out: {
		if (IN & string) != _|_ {
			[
				{#ToProbeSpec & {in: {def: IN, type: "readiness"}}}.out,
			]
		}
		if (IN & v1.#ProbeMap) != _|_ {
			[ for name, probe in IN {
				if name == "ready" {
					let t = "readiness"
					{#ToProbeSpec & {in: {def: probe, type: t}}}.out
				}
				if name == "readiness" {
					let t = "readiness"
					{#ToProbeSpec & {in: {def: probe, type: t}}}.out
				}
				if name == "liveness" {
					let t = "liveness"
					{#ToProbeSpec & {in: {def: probe, type: t}}}.out
				}
				if name == "startup" {
					let t = "startup"
					{#ToProbeSpec & {in: {def: probe, type: t}}}.out
				}
			}]
		}
		if (IN & [...v1.#ProbeSpec]) != _|_ {
			IN
		}
	}
}
