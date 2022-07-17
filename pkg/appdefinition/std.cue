import (
	_std_list "list"
	_std_strings "strings"
	_std_net "net"
	_std_yaml "encoding/yaml"
	_std_json "encoding/json"
	_std_hex "encoding/hex"
	_std_base64 "encoding/base64"
	_std_sha1 "crypto/sha1"
	_std_sha256 "crypto/sha256"
	_std_sha512 "crypto/sha512"
	_std_path "path"
	_std_strconv "strconv"
	_std_tabwriter "text/tabwriter"
)

let std = {

	atoi: {
		_args: [string]
		out: int
		out: _std_strconv.Atoi(_args[0])
	}

	fileExt: {
		_args: [string]
		out: string
		out: _std_path.Ext(_args[0])
	}

	basename: {
		_args: [string]
		out: string
		out: _std_path.Base(_args[0])
	}

	dirname: {
		_args: [string]
		out: string
		out: _std_path.Dir(_args[0])
	}

	pathJoin: {
		_args: [[...string], string] | [[...string]]
		out:   string
		if len(_args) == 1 {
			out: _std_path.Join(_args[0], "unix")
		}
		if len(_args) == 2 {
			if _args[1] == "/" {
				out: _std_path.Join(_args[0], "unix")
			}
			if _args[1] == "\\" {
				out: _std_path.Join(_args[0], "windows")
			}
			if _args[1] != "\\" && _args[1] != "/" {
				out: _std_path.Join(_args[0], _args[1])
			}
		}
	}

	splitHostPort: {
		_args: [string]
		out: [...string]
		out: _std_net.SplitHostPort(_args[0])
	}

	joinHostPort: {
		_args: [string, string | int]
		out: string
		out: _std_net.JoinHostPort(_args[0], _args[1])
	}

	base64decode: {
		_args: [string]
		out: bytes
		out: _std_base64.Decode(null, _args[0])
	}

	base64: {
		_args: [bytes | string]
		out: string
		out: _std_base64.Encode(null, _args[0])
	}

	sha1sum: {
		_args: [bytes | string]
		out: string
		out: _std_hex.Encode(_std_sha1.Sum(_args[0]))
	}

	sha256sum: {
		_args: [bytes | string]
		out: string
		out: _std_hex.Encode(_std_sha256.Sum256(_args[0]))
	}

	sha512sum: {
		_args: [bytes | string]
		out: string
		out: _std_hex.Encode(_std_sha512.Sum512(_args[0]))
	}

	toHex: {
		_args: [bytes | string]
		out: string
		out: _std_hex.Encode(_args[0])
	}

	fromHex: {
		_args: [string]
		out: bytes | string
		out: _std_hex.Decode(_args[0])
	}

	toJSON: {
		_args: [_]
		out: string
		out: _std_json.Marshal(_args[0])
	}

	fromJSON: {
		_args: [string | bytes]
		out: _
		out: _std_json.Unmarshal(_args[0])
	}

	toYAML: {
		_args: [_]
		out: _
		out: _std_yaml.Marshal(_args[0])
	}

	fromYAML: {
		_args: [bytes | string]
		out: _
		out: _std_yaml.Unmarshal(_args[0])
	}

	ifelse: {
		_args: [bool, _, _]
		out: _
		if _args[0] {
			out: _args[1]
		}
		if !_args[0] {
			out: _args[2]
		}
	}

	reverse: {
		_args: [[...]]
		out: [...]
		out: [ for i in _std_list.Range(len(_args[0])-1, -1, -1) {
			_args[0][i]
		}]
	}

	sort: {
		_args: [[...], {
			T:    _
			x:    T
			y:    T
			less: bool
		}] | [[...]]
		out: [...]
		if len(_args) == 1 {
			out: _std_list.Sort(_args[0], _std_list.Ascending)
		}
		if len(_args) == 2 {
			out: _std_list.Sort(_args[0], _args[1])
		}
	}

	slice: {
		_args: [[...], int, int]
		out: [...]
		out: _std_list.Slice(_args[0], _args[1], _args[2])
	}

	range: {
		_args: [int, int, int] | [int, int] | [int]
		out: [...int]
		if len(_args) == 1 {
			out: _std_list.Range(0, _args[0], 1)
		}
		if len(_args) == 2 {
			out: _std_list.Range(_args[0], _args[1], 1)
		}
		if len(_args) == 3 {
			out: _std_list.Range(_args[0], _args[1], _args[2])
		}
	}

	toTitle: {
		_args: [string]
		out: string
		out: _std_strings.ToTitle(_args[0])
	}

	contains: {
		_args: [string, string] | [[...], _] | [ {}, string]
		out:   bool

		if (_args[0] & string) != _|_ {
			out: _std_strings.Contains(_args[0], _args[1])
		}
		if (_args[0] & [...]) != _|_ {
			out: _std_list.Contains(_args[0], _args[1])
		}
		if (_args[0] & {}) != _|_ {
			out: bool | *false
			if (_args[0] & _args[1]) != _|_ {
				out: true
			}
		}
	}

	split: {
		_args: [string, string, int] | [string, string]
		out: [...string]
		if len(_args) == 3 {
			out: _std_strings.SplitN(_args[0], _args[1], _args[2])
		}
		if len(_args) == 2 {
			out: _std_strings.SplitN(_args[0], _args[1], -1)
		}
	}

	join: {
		_args: [[...string], string]
		out: string
		out: _std_strings.Join(_args[0], _args[1])
	}

	endsWith: {
		_args: [string, string]
		out: bool
		out: _std_strings.HasSuffix(_args[0], _args[1])
	}

	startsWith: {
		_args: [string, string]
		out: bool
		out: _std_strings.HasPrefix(_args[0], _args[1])
	}

	toUpper: {
		_args: [string]
		out: string
		out: _std_strings.ToUpper(_args[0])
	}

	toLower: {
		_args: [string]
		out: string
		out: _std_strings.ToLower(_args[0])
	}

	trim: {
		_args: [string]
		out: string
		out: _std_strings.TrimSpace(_args[0])
	}

	trimSuffix: {
		_args: [string, string]
		out: string
		out: _std_strings.TrimSuffix(_args[0], _args[1])
	}

	trimPrefix: {
		_args: [string, string]
		out: string
		out: _std_strings.TrimPrefix(_args[0], _args[1])
	}

	replace: {
		_args: [string, string, string, int] | [string, string, string]
		out:   string
		if len(_args) == 3 {
			out: _std_strings.Replace(_args[0], _args[1], _args[2], -1)
		}
		if len(_args) == 4 {
			out: _std_strings.Replace(_args[0], _args[1], _args[2], _args[3])
		}
	}

	indexOf: {
		_args: [string, string] | [[...], _]
		out:   int
		if (_args[0] & string) != _|_ {
			out: _std_strings.Index(_args[0], _args[1])
		}
		if (_args[0] & [...]) != _|_ {
			out: int | -1
			for i, v in _args[0] {
				if v == _args[1] {
					out: i
				}
			}
		}
	}

	merge: {
		_args: [{}, {}]
		out: {}

		let left = _args[0]
		let right = _args[1]
		out: {
			for k, lv in left {
				let rv = right[k]
				if rv != _|_ {
					// exists in right
					if (rv & {}) != _|_ {
						// is map so merge
						"\(k)": (merge & {_args: [lv, rv]}).out
					}
					if !((rv & {}) != _|_) {
						// is map so merge
						"\(k)": rv
					}
				}
				if !(rv != _|_) {
					// does not exists in right
					"\(k)": lv
				}
			}
			for k, v in right {
				if !(left[k] != _|_) {
					"\(k)": v
				}
			}
		}
	}

}