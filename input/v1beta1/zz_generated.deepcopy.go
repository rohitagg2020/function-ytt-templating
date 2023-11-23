//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplateSourceFileSystem) DeepCopyInto(out *TemplateSourceFileSystem) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplateSourceFileSystem.
func (in *TemplateSourceFileSystem) DeepCopy() *TemplateSourceFileSystem {
	if in == nil {
		return nil
	}
	out := new(TemplateSourceFileSystem)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *YTT) DeepCopyInto(out *YTT) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.FileSystem != nil {
		in, out := &in.FileSystem, &out.FileSystem
		*out = new(TemplateSourceFileSystem)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new YTT.
func (in *YTT) DeepCopy() *YTT {
	if in == nil {
		return nil
	}
	out := new(YTT)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *YTT) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
