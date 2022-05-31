package normalize

import (
	"github.com/acorn-io/acorn/schema/v1"
	"strings"
	"encoding/base64"
)

#ToFileSpec: {
	IN="in": string | bytes
	out:     v1.#FileSpec
	out: {
		let switch = {
			"bytes": {
				input: bytes & IN
				out: {
					content: base64.Encode(null, input)
				}
			}
			"string": {
				input: string & IN
				out: {
					if strings.HasPrefix(input, "secret://") {
						let _uri = {#ToURI & {in: input}}.out
						secret: {
							name: _uri.name
							key:  _uri.path
							if _uri.query["onchange"] != _|_ {
								if _uri.query["onchange"][0] == "no-action" {
									onChange: "noAction"
								}
							}
						}
					}
					if !strings.HasPrefix(input, "secret://") {
						content: base64.Encode(null, input)
					}
				}
			}
		}

		switch.bytes.out | switch.string.out
	}
}
