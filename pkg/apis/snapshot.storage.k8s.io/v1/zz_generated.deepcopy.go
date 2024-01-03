//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshot) DeepCopyInto(out *VolumeSnapshot) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(VolumeSnapshotStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshot.
func (in *VolumeSnapshot) DeepCopy() *VolumeSnapshot {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshot) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotClass) DeepCopyInto(out *VolumeSnapshotClass) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotClass.
func (in *VolumeSnapshotClass) DeepCopy() *VolumeSnapshotClass {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotClass)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshotClass) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotClassList) DeepCopyInto(out *VolumeSnapshotClassList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeSnapshotClass, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotClassList.
func (in *VolumeSnapshotClassList) DeepCopy() *VolumeSnapshotClassList {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotClassList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshotClassList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotContent) DeepCopyInto(out *VolumeSnapshotContent) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(VolumeSnapshotContentStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotContent.
func (in *VolumeSnapshotContent) DeepCopy() *VolumeSnapshotContent {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotContent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshotContent) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotContentList) DeepCopyInto(out *VolumeSnapshotContentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeSnapshotContent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotContentList.
func (in *VolumeSnapshotContentList) DeepCopy() *VolumeSnapshotContentList {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotContentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshotContentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotContentSource) DeepCopyInto(out *VolumeSnapshotContentSource) {
	*out = *in
	if in.VolumeHandle != nil {
		in, out := &in.VolumeHandle, &out.VolumeHandle
		*out = new(string)
		**out = **in
	}
	if in.SnapshotHandle != nil {
		in, out := &in.SnapshotHandle, &out.SnapshotHandle
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotContentSource.
func (in *VolumeSnapshotContentSource) DeepCopy() *VolumeSnapshotContentSource {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotContentSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotContentSpec) DeepCopyInto(out *VolumeSnapshotContentSpec) {
	*out = *in
	out.VolumeSnapshotRef = in.VolumeSnapshotRef
	if in.VolumeSnapshotClassName != nil {
		in, out := &in.VolumeSnapshotClassName, &out.VolumeSnapshotClassName
		*out = new(string)
		**out = **in
	}
	in.Source.DeepCopyInto(&out.Source)
	if in.SourceVolumeMode != nil {
		in, out := &in.SourceVolumeMode, &out.SourceVolumeMode
		*out = new(corev1.PersistentVolumeMode)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotContentSpec.
func (in *VolumeSnapshotContentSpec) DeepCopy() *VolumeSnapshotContentSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotContentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotContentStatus) DeepCopyInto(out *VolumeSnapshotContentStatus) {
	*out = *in
	if in.SnapshotHandle != nil {
		in, out := &in.SnapshotHandle, &out.SnapshotHandle
		*out = new(string)
		**out = **in
	}
	if in.CreationTime != nil {
		in, out := &in.CreationTime, &out.CreationTime
		*out = new(int64)
		**out = **in
	}
	if in.RestoreSize != nil {
		in, out := &in.RestoreSize, &out.RestoreSize
		*out = new(int64)
		**out = **in
	}
	if in.ReadyToUse != nil {
		in, out := &in.ReadyToUse, &out.ReadyToUse
		*out = new(bool)
		**out = **in
	}
	if in.Error != nil {
		in, out := &in.Error, &out.Error
		*out = new(VolumeSnapshotError)
		(*in).DeepCopyInto(*out)
	}
	if in.VolumeGroupSnapshotHandle != nil {
		in, out := &in.VolumeGroupSnapshotHandle, &out.VolumeGroupSnapshotHandle
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotContentStatus.
func (in *VolumeSnapshotContentStatus) DeepCopy() *VolumeSnapshotContentStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotContentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotError) DeepCopyInto(out *VolumeSnapshotError) {
	*out = *in
	if in.Time != nil {
		in, out := &in.Time, &out.Time
		*out = (*in).DeepCopy()
	}
	if in.Message != nil {
		in, out := &in.Message, &out.Message
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotError.
func (in *VolumeSnapshotError) DeepCopy() *VolumeSnapshotError {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotError)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotList) DeepCopyInto(out *VolumeSnapshotList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeSnapshot, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotList.
func (in *VolumeSnapshotList) DeepCopy() *VolumeSnapshotList {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeSnapshotList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotSource) DeepCopyInto(out *VolumeSnapshotSource) {
	*out = *in
	if in.PersistentVolumeClaimName != nil {
		in, out := &in.PersistentVolumeClaimName, &out.PersistentVolumeClaimName
		*out = new(string)
		**out = **in
	}
	if in.VolumeSnapshotContentName != nil {
		in, out := &in.VolumeSnapshotContentName, &out.VolumeSnapshotContentName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotSource.
func (in *VolumeSnapshotSource) DeepCopy() *VolumeSnapshotSource {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotSpec) DeepCopyInto(out *VolumeSnapshotSpec) {
	*out = *in
	in.Source.DeepCopyInto(&out.Source)
	if in.VolumeSnapshotClassName != nil {
		in, out := &in.VolumeSnapshotClassName, &out.VolumeSnapshotClassName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotSpec.
func (in *VolumeSnapshotSpec) DeepCopy() *VolumeSnapshotSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSnapshotStatus) DeepCopyInto(out *VolumeSnapshotStatus) {
	*out = *in
	if in.BoundVolumeSnapshotContentName != nil {
		in, out := &in.BoundVolumeSnapshotContentName, &out.BoundVolumeSnapshotContentName
		*out = new(string)
		**out = **in
	}
	if in.CreationTime != nil {
		in, out := &in.CreationTime, &out.CreationTime
		*out = (*in).DeepCopy()
	}
	if in.ReadyToUse != nil {
		in, out := &in.ReadyToUse, &out.ReadyToUse
		*out = new(bool)
		**out = **in
	}
	if in.RestoreSize != nil {
		in, out := &in.RestoreSize, &out.RestoreSize
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Error != nil {
		in, out := &in.Error, &out.Error
		*out = new(VolumeSnapshotError)
		(*in).DeepCopyInto(*out)
	}
	if in.VolumeGroupSnapshotName != nil {
		in, out := &in.VolumeGroupSnapshotName, &out.VolumeGroupSnapshotName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSnapshotStatus.
func (in *VolumeSnapshotStatus) DeepCopy() *VolumeSnapshotStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeSnapshotStatus)
	in.DeepCopyInto(out)
	return out
}
