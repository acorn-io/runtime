package v1

import (
	"net/url"
	"unsafe"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
)

func convertURLlValuesToContainerReplicaExecOptions(in *url.Values, out *ContainerReplicaExecOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["command"]; ok && len(values) > 0 {
		out.Command = *(*[]string)(unsafe.Pointer(&values))
	} else {
		out.Command = nil
	}
	if values, ok := map[string][]string(*in)["tty"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.TTY, s); err != nil {
			return err
		}
	} else {
		out.TTY = false
	}
	if values, ok := map[string][]string(*in)["debugImage"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.DebugImage, s); err != nil {
			return err
		}
	} else {
		out.DebugImage = ""
	}
	return nil
}

func ConvertURLValuesToContainerReplicaExecOptions(in, out interface{}, s conversion.Scope) error {
	return convertURLlValuesToContainerReplicaExecOptions(in.(*url.Values), out.(*ContainerReplicaExecOptions), s)
}

func convertURLValuesToLogOptions(in *url.Values, out *LogOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["tailLines"]; ok && len(values) > 0 {
		out.Tail = new(int64)
		if err := runtime.Convert_Slice_string_To_int64(&values, out.Tail, s); err != nil {
			return err
		}
	}
	if values, ok := map[string][]string(*in)["follow"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.Follow, s); err != nil {
			return err
		}
	}
	if values, ok := map[string][]string(*in)["containerReplica"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.ContainerReplica, s); err != nil {
			return err
		}
	}
	if values, ok := map[string][]string(*in)["container"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.Container, s); err != nil {
			return err
		}
	}
	return nil
}

func ConvertURLValuesToLogOptions(in, out interface{}, s conversion.Scope) error {
	return convertURLValuesToLogOptions(in.(*url.Values), out.(*LogOptions), s)
}

func convertURLValuesToContainerReplicaPortForwardOptions(in *url.Values, out *ContainerReplicaPortForwardOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["port"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_int(&values, &out.Port, s); err != nil {
			return err
		}
	} else {
		out.Port = 0
	}
	return nil
}

func ConvertURLValuesToContainerReplicaPortForwardOptions(in, out interface{}, s conversion.Scope) error {
	return convertURLValuesToContainerReplicaPortForwardOptions(in.(*url.Values), out.(*ContainerReplicaPortForwardOptions), s)
}
