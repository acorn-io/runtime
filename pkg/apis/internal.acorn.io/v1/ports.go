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

func parseSingle(str string) (int32, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid port syntax [%s]: %w", str, err)
	}
	return int32(i), nil
}

func parseQuad(expose bool, left, leftMiddle, rightMiddle, right string) (PortBinding, error) {
	if !expose {
		return PortBinding{}, fmt.Errorf("invalid [%s:%s:%s:%s]: (service:port:service:port) syntax"+
			" is only valid for expose", left, leftMiddle, rightMiddle, right)
	}
	_, leftIsNum, err := parseNum(left)
	if err != nil {
		return PortBinding{}, err
	}
	leftMiddleNum, leftMiddleIsNum, err := parseNum(leftMiddle)
	if err != nil {
		return PortBinding{}, err
	}
	_, rightMiddleIsNum, err := parseNum(rightMiddle)
	if err != nil {
		return PortBinding{}, err
	}
	rightNum, rightIsNum, err := parseNum(right)
	if err != nil {
		return PortBinding{}, err
	}

	if !leftIsNum && leftMiddleIsNum && !rightMiddleIsNum && rightIsNum {
		return PortBinding{
			ServiceName:       left,
			Port:              leftMiddleNum,
			TargetPort:        rightNum,
			TargetServiceName: rightMiddle,
		}, nil
	}
	return PortBinding{}, fmt.Errorf("invalid [%s:%s:%s:%s]: must be (service:port:service:port)",
		left, leftMiddle, rightMiddle, right)
}

func parseTriplet(left, middle, right string) (PortBinding, error) {
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
			ServiceName:       left,
			TargetPort:        rightNum,
			TargetServiceName: middle,
		}, nil
	}
	return PortBinding{}, fmt.Errorf("invalid binding [%s:%s:%s] must be service:port:targetPort or domain:service:targetPort", left, middle, right)
}

func parseTuple(binding bool, left, right string) (PortBinding, error) {
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
		// service:80 format
		if binding {
			return PortBinding{
				TargetPort:        rightNum,
				TargetServiceName: left,
			}, nil
		} else {
			return PortBinding{
				ServiceName: left,
				TargetPort:  rightNum,
			}, nil
		}
	} else if leftIsNum && !rightIsNum {
		// 80:service format
		return PortBinding{}, fmt.Errorf("invalidate port binding [%s:%s] can not be number:string format", left, right)
	}
	// example.com:name
	return PortBinding{
		ServiceName:       left,
		TargetServiceName: right,
		Protocol:          ProtocolHTTP,
	}, nil
}

func ParsePorts(args []string) (result []PortDef, _ error) {
	pbs, err := parseBindings(false, false, args)
	if err != nil {
		return nil, err
	}
	for _, pb := range pbs {
		pb.Expose = false
		pb.Publish = false
		result = append(result, (PortDef)(pb.Complete("")))
	}
	return
}

func ParsePortBindings(publish bool, args []string) (result []PortBinding, _ error) {
	return parseBindings(true, publish, args)
}

func parseBindings(isBinding, publish bool, args []string) (result []PortBinding, _ error) {
	for _, arg := range args {
		var (
			binding PortBinding
			err     error
		)

		arg, proto, _ := strings.Cut(arg, "/")
		parts := strings.Split(arg, ":")

		switch len(parts) {
		case 1:
			binding.TargetPort, err = parseSingle(parts[0])
			if err != nil {
				return nil, err
			}
		case 2:
			binding, err = parseTuple(isBinding, parts[0], parts[1])
			if err != nil {
				return nil, err
			}
		case 3:
			binding, err = parseTriplet(parts[0], parts[1], parts[2])
			if err != nil {
				return nil, err
			}
		case 4:
			binding, err = parseQuad(!publish, parts[0], parts[1], parts[2], parts[3])
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid syntax [%s] too many colon separated parts", arg)
		}

		if p, ok := validProto(proto); !ok {
			return nil, fmt.Errorf("invalid protocol [%s]", p)
		} else if binding.Protocol != "" && p != "" && binding.Protocol != p {
			return nil, fmt.Errorf("inferred protocol [%s] does not match requested protocol [%s]", binding.Protocol, p)
		} else if binding.Protocol == "" {
			binding.Protocol = p
		}

		binding.Publish = publish
		binding.Expose = !publish

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

func ParseVolumes(args []string, binding bool) (result []VolumeBinding, _ error) {
	for _, arg := range args {
		arg, opts, _ := strings.Cut(arg, ",")
		existing, volName, ok := strings.Cut(arg, ":")
		if !ok {
			volName = existing
			if binding {
				// In a binding no existing means we want to configure the generated volume, not bind one
				existing = ""
			}
		}
		volName = strings.TrimSpace(volName)
		existing = strings.TrimSpace(existing)
		if volName == "" {
			return nil, fmt.Errorf("invalid endpoint binding [%s] must not have zero length value", arg)
		}
		volumeBinding := VolumeBinding{
			Volume: existing,
			Target: volName,
		}

		if binding {
			opts := KVMap(opts, ",")
			volumeBinding.Class = strings.TrimSpace(opts["class"])
			q, err := ParseQuantity(opts["size"])
			if err != nil {
				return nil, fmt.Errorf("parsing [%s]: %w", arg, err)
			}
			volumeBinding.Size = q
		}

		result = append(result, volumeBinding)
	}
	return
}
