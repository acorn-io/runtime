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
						let uri = {#ToURI & {in: input}}.out
						secret: {
							name: uri.name
							key:  uri.path
							if uri.query["optional"][0] != _|_ {
								optional: uri.query.optional[0] == "true"
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
