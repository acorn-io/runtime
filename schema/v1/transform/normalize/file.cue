package normalize

import (
	"github.com/acorn-io/acorn/schema/v1"
	"strings"
	"encoding/base64"
)

#ToFileSpec: {
	IN="in": {
		key:   string
		value: string | bytes | v1.#FileSpec
	}
	out: v1.#FileSpec
	out: {
		let switch = {
			"struct": {
				out: IN.value
			}
			"bytes": {
				input: bytes & IN.value
				out: {
					content: base64.Encode(null, input)
					if IN.key =~ ".*/bin/.*|.*/sbin/.*|\\.sh$" {
						mode: "0755"
					}
				}
			}
			"string": {
				input: string & IN.value
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
						if _uri.query["mode"] != _|_ {
							mode: _uri.query["mode"][0]
						}
						if !(_uri.query["mode"] != _|_) {
							if IN.key =~ ".*/bin/.*|.*/sbin/.*|\\.sh$" {
								mode: "0755"
							}
						}
					}
					if !strings.HasPrefix(input, "secret://") {
						content: base64.Encode(null, input)
						if IN.key =~ ".*/bin/.*|.*/sbin/.*|\\.sh$" {
							mode: "0755"
						}
					}
				}
			}
		}

		switch.bytes.out | switch.string.out | switch.struct.out
	}
}
