package normalize

import (
	"list"
	"strings"
	"github.com/ibuildthecloud/herd/schema/v1"
)

#ToEnvVarSpecFromString: {
	IN="in": {
		name:  string
		value: string
	}
	out: v1.#EnvVarSpec
	out: {
		let switch = {
			secretembed: {
				input: {
					name:  =~"^secret://.*"
					value: ""
				} & IN
				out: {
					let _uri = {#ToURI & {in: input.name}}.out
					name: ""
					secret: {
						name: _uri.name
						key:  _uri.path
						if _uri.query["optional"][0] != _|_ {
							optional: _uri.query["optional"][0] == "true"
						}
					}

				}
			}
			secretvalue: {
				input: {
					name:  string
					value: =~"^secret://.*"
				} & IN
				out: {
					let _uri = {#ToURI & {in: input.value}}.out
					name: input.name
					secret: {
						name: _uri.name
						key:  _uri.path
						if _uri.query["optional"][0] != _|_ {
							optional: _uri.query["optional"][0] == "true"
						}
					}
				}
			}
			keyvalue: {
				input: {
					name:  !~"^secret://.*|^$"
					value: !~"^secret://.*|^$"
				} & IN
				out: {
					name:  input.name
					value: input.value
				}
			}
		}
		switch.secretembed.out | switch.secretvalue.out | switch.keyvalue.out
	}
}

#ToEnvFromStrings: {
	IN="in": [...string]
	out: [...v1.#EnvVarSpec]

	out: [ for v in IN {
		if strings.HasPrefix(v, "secret://") {
			{#ToEnvVarSpecFromString & {in: {
				name:  v
				value: ""
			}}}.out
		}
		if !strings.HasPrefix(v, "secret://") {
			let _parts = strings.SplitN(v, "=", 2)
			{#ToEnvVarSpecFromString & {in: {
				name:  _parts[0]
				value: _parts[1] | *""
			}}}.out
		}
	}]
}

#ToEnvFromMap: {
	IN="in": [string]: string
	out: [...v1.#EnvVarSpec]

	out: [ for k, v in IN {
		{#ToEnvVarSpecFromString & {in: {
			name:  k
			value: v
		}}}.out
	}]
}

#ToEnvVarSpecs: {
	IN="in": v1.#EnvVars
	out: [...v1.#EnvVarSpec]
	out: list.Sort({#ToEnvFromStrings & {in: IN}}.out | {#ToEnvFromMap & {in: IN}}.out, {
		x: {}, y: {}
		less: {
			let left = strings.Join([x.name, x.secret.name | *"", x.secret.key | *""], "::")
			let right = strings.Join([y.name, y.secret.name | *"", y.secret.key | *""], "::")
			left < right
		}
	})
}
