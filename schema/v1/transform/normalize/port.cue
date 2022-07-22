package normalize

import (
	"github.com/acorn-io/acorn/schema/v1"
	"regexp"
	"strings"
	"strconv"
	"list"
)

#ToPortFromString: {
	IN="in": string
	out:     v1.#PortSpec
	out: {
		let _m = regexp.FindNamedSubmatch(v1.PortRegexp, IN)
		targetPort: strconv.Atoi(_m["targetPort"])
		if _m["port"] != "" {
			port: strconv.Atoi(strings.TrimSuffix(_m["port"], ":"))
		}
		if _m["proto"] != "" {
			protocol: strings.TrimPrefix(_m["proto"], "/")
		}
		if _m["serviceName"] == "" && _m["targetServiceName"] != "" && _m["port"] == "" {
			serviceName: strings.TrimSuffix(_m["targetServiceName"], ":")
		}
		if !(_m["serviceName"] == "" && _m["targetServiceName"] != "" && _m["port"] == "") {
			serviceName:       strings.TrimSuffix(_m["serviceName"], ":")
			targetServiceName: strings.TrimSuffix(_m["targetServiceName"], ":")
		}
	}
}

#ToPort: {
	IN="in": _
	out:     v1.#PortSpec
	out: {
		if (IN & string) != _|_ {(#ToPortFromString & {in: IN}).out}
		if (IN & int) != _|_ {targetPort: IN}
		if !((IN & string) != _|_ ) && !((IN & int) != _|_) {IN}
	}
}

#ToPortsFromMap: {
	IN="in": _
	mode:    string
	out: [...v1.#PortSpec]
	out: {
		if !(IN[mode] != _|_) {
			[]
		}
		if IN[mode] != _|_ {
			if (IN[mode] & int) != _|_ || (IN[mode] & string) != _|_ {
				[{
					(#ToPort & {in: IN[mode]}).out
					if mode != "internal" {
						"\(mode)": true
					}
				}]
			}
			if !((IN[mode] & int) != _|_) &&
				!((IN[mode] & string) != _|_) {
				[ for x in IN[mode] {
					(#ToPort & {in: x}).out
					if mode != "internal" {
						"\(mode)": true
					}
				}, ...]
			}
		}
	}
}

#ToPortsFromList: {
	IN="in": _
	out: [...v1.#PortSpec]
	out: {
		if IN[0] != _|_ {
			[ for x in IN {(#ToPort & {in: x}).out}]
		}
	}
}

#ToPorts: {
	IN="in": _ // removed because too slow: v1.#Port | *[...v1.#Port] | v1.#PortMap
	out: [...v1.#PortSpec]
	out: {
		if (IN & int) != _|_ {
			[{targetPort: IN}]
		}
		if (IN & string) != _|_ {
			[(#ToPortFromString & {in: IN}).out]
		}
		let modeList = list.Concat([
			(#ToPortsFromMap & {in: IN, mode: "expose"}).out,
			(#ToPortsFromMap & {in: IN, mode: "internal"}).out,
			(#ToPortsFromMap & {in: IN, mode: "publish"}).out,
		])
		if len(modeList) > 0 {
			modeList
		}
		let portsList = {(#ToPortsFromList & {in: IN}).out}
		if len(portsList) > 0 {
			portsList
		}
	}
}
