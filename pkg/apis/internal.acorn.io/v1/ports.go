package v1

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	nameRegexp = regexp.MustCompile("^[a-z][-a-z0-9.]+$")
)

func validProto(p string) (Protocol, bool) {
	ret := Protocol(p)
	switch ret {
	case ProtocolTCP:
		fallthrough
	case ProtocolUDP:
		fallthrough
	case ProtocolHTTP:
		return ret, true
	case "":
		return ret, true
	}
	return ret, false
}

func parseNum(str string) (int32, bool, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		if !nameRegexp.MatchString(str) {
			return 0, false, fmt.Errorf("string [%s] does not match %s", str, nameRegexp)
		}
		return 0, false, nil
	}
	return int32(i), true, nil
}

func parsePortBindingSingle(str string) (string, int32, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		if !nameRegexp.MatchString(str) {
			return "", 0, fmt.Errorf("string [%s] does not match %s", str, nameRegexp)
		}
		return str, 0, nil
	}
	return "", int32(i), nil
}

func parsePortSingle(str string) (int32, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid port syntax [%s]: %w", str, err)
	}
	return int32(i), nil
}

func parsePortTriplet(left, middle, right string) (PortDef, error) {
	_, leftIsNum, err := parseNum(left)
	if err != nil {
		return PortDef{}, err
	}
	middleNum, middleIsNum, err := parseNum(middle)
	if err != nil {
		return PortDef{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return PortDef{}, err
	}

	if !leftIsNum && middleIsNum && rightIsNum {
		// hostname:81:80
		return PortDef{
			Port:       middleNum,
			TargetPort: rightNum,
			Hostname:   left,
			Protocol:   ProtocolHTTP,
		}, nil
	}
	return PortDef{}, fmt.Errorf("invalid binding [%s:%s:%s] must be [hostname:port:port]", left, middle, right)
}

func parsePortBindingTriplet(left, middle, right string) (PortBinding, error) {
	leftNum, leftIsNum, err := parseNum(left)
	if err != nil {
		return PortBinding{}, err
	}
	_, middleIsNum, err := parseNum(middle)
	if err != nil {
		return PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return PortBinding{}, err
	}

	if leftIsNum && !middleIsNum && rightIsNum {
		// 81:service:80
		return PortBinding{
			Port:              leftNum,
			TargetPort:        rightNum,
			TargetServiceName: middle,
		}, nil
	} else if !leftIsNum && !middleIsNum && rightIsNum {
		// example.com:service:80
		return PortBinding{
			Hostname:          left,
			Protocol:          ProtocolHTTP,
			TargetPort:        rightNum,
			TargetServiceName: middle,
		}, nil
	}
	return PortBinding{}, fmt.Errorf("invalid binding [%s:%s:%s] must be [hostname:service:port] or [port:service:port]", left, middle, right)
}

func parsePortBindingTuple(left, right string) (PortBinding, error) {
	leftNum, leftIsNum, err := parseNum(left)
	if err != nil {
		return PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return PortBinding{}, err
	}

	if leftIsNum && rightIsNum {
		// 81:80 format
		return PortBinding{
			Port:       leftNum,
			TargetPort: rightNum,
		}, nil
	} else if !leftIsNum && rightIsNum {
		if strings.Contains(left, ".") {
			// example.com:80 format
			return PortBinding{
				Hostname:   left,
				TargetPort: rightNum,
				Protocol:   ProtocolHTTP,
			}, nil
		} else {
			// app:80 format
			return PortBinding{
				TargetServiceName: left,
				TargetPort:        rightNum,
			}, nil
		}
	} else if leftIsNum && !rightIsNum {
		// 80:service format
		return PortBinding{
			Port:              leftNum,
			TargetServiceName: right,
		}, nil
	} else {
		if strings.Contains(left, ".") {
			// hostname:service
			return PortBinding{
				Hostname:          left,
				TargetServiceName: right,
				Protocol:          ProtocolHTTP,
			}, nil
		}
		return PortBinding{}, fmt.Errorf("[%s] must be a hostname containing a \".\"", left)
	}
}

func parsePortTuple(left, right string) (PortDef, error) {
	leftNum, leftIsNum, err := parseNum(left)
	if err != nil {
		return PortDef{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return PortDef{}, err
	}

	if leftIsNum && rightIsNum {
		// 81:80 format
		return PortDef{
			Port:       leftNum,
			TargetPort: rightNum,
		}, nil
	} else if !leftIsNum && rightIsNum {
		// example.com:80 format
		if !strings.Contains(left, ".") {
			return PortDef{}, fmt.Errorf("[%s] is not a valid hostname to publish, missing \".\"", left)
		}
		return PortDef{
			Hostname:   left,
			TargetPort: rightNum,
			Protocol:   ProtocolHTTP,
		}, nil
	}

	return PortDef{}, fmt.Errorf("invalidate port [%s:%s] must be [hostname:port] or [port:port] format", left, right)
}

func ParsePorts(args []string) (result []PortDef, _ error) {
	for _, arg := range args {
		var (
			port PortDef
			err  error
		)

		arg, proto, _ := strings.Cut(arg, "/")
		parts := strings.Split(arg, ":")

		switch len(parts) {
		case 1:
			port.TargetPort, err = parsePortSingle(parts[0])
			if err != nil {
				return nil, err
			}
		case 2:
			port, err = parsePortTuple(parts[0], parts[1])
			if err != nil {
				return nil, err
			}
		case 3:
			port, err = parsePortTriplet(parts[0], parts[1], parts[2])
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid syntax [%s] too many colon separated parts", arg)
		}

		if p, ok := validProto(proto); !ok {
			return nil, fmt.Errorf("invalid protocol [%s]", p)
		} else if port.Protocol != "" && p != "" && port.Protocol != p {
			return nil, fmt.Errorf("inferred protocol [%s] does not match requested protocol [%s]", port.Protocol, p)
		} else if port.Protocol == "" {
			port.Protocol = p
		}

		result = append(result, port)
	}
	return
}

func ParsePortBindings(args []string) (result []PortBinding, _ error) {
	for _, arg := range args {
		var (
			binding PortBinding
			err     error
		)

		arg, proto, _ := strings.Cut(arg, "/")
		parts := strings.Split(arg, ":")

		switch len(parts) {
		case 1:
			binding.TargetServiceName, binding.TargetPort, err = parsePortBindingSingle(parts[0])
			if err != nil {
				return nil, err
			}
		case 2:
			binding, err = parsePortBindingTuple(parts[0], parts[1])
			if err != nil {
				return nil, err
			}
			if (binding.Protocol == ProtocolHTTP || (binding.Protocol == "" && proto == string(ProtocolHTTP))) &&
				binding.Port != 0 {
				return nil, fmt.Errorf("can not bind an http port [%d] to an alternative port [%d], only hostname", binding.TargetPort, binding.Port)
			}
		case 3:
			binding, err = parsePortBindingTriplet(parts[0], parts[1], parts[2])
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid syntax [%s] too many colon separated parts, only 0-2 colons allowed got [%d]", arg, len(parts)-1)
		}

		if p, ok := validProto(proto); !ok {
			return nil, fmt.Errorf("invalid protocol [%s]", p)
		} else if binding.Protocol != "" && p != "" && binding.Protocol != p {
			return nil, fmt.Errorf("inferred protocol [%s] does not match requested protocol [%s]", binding.Protocol, p)
		} else if binding.Protocol == "" {
			binding.Protocol = p
		}

		result = append(result, binding)
	}
	return
}

func ParseLinks(args []string) (result []ServiceBinding, _ error) {
	for _, arg := range args {
		existing, secName, ok := strings.Cut(arg, ":")
		if !ok {
			secName = existing
		}
		secName = strings.TrimSpace(secName)
		existing = strings.TrimSpace(existing)
		if secName == "" || existing == "" {
			return nil, fmt.Errorf("invalid service binding [%s] must not have zero length value", arg)
		}
		result = append(result, ServiceBinding{
			Target:  secName,
			Service: existing,
		})
	}
	return
}

func ParseSecrets(args []string) (result []SecretBinding, _ error) {
	for _, arg := range args {
		existing, secName, ok := strings.Cut(arg, ":")
		if !ok {
			secName = existing
		}
		secName = strings.TrimSpace(secName)
		existing = strings.TrimSpace(existing)
		if secName == "" || existing == "" {
			return nil, fmt.Errorf("invalid endpoint binding [%s] must not have zero length value", arg)
		}
		result = append(result, SecretBinding{
			Secret: existing,
			Target: secName,
		})
	}
	return
}

func KVMap(val string, sep string) map[string]string {
	result := map[string]string{}
	for _, part := range strings.Split(val, sep) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		result[k] = v
	}
	return result
}
