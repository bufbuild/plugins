// Copyright 2020-2023 Buf Technologies, Inc.
//
// All rights reserved.

package bufwasmtool

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoEnum interface {
	~int32
	Descriptor() protoreflect.EnumDescriptor
}

// NewProtoEnumFlag turns a proto enum into a cobra flag.
func NewProtoEnumFlag[T ProtoEnum]() *protoEnumFlag[T] {
	var newT T
	enumDescriptor := newT.Descriptor()
	names := make(map[int32]string, enumDescriptor.Values().Len())
	values := make(map[string]int32, enumDescriptor.Values().Len())
	allowed := make([]string, enumDescriptor.Values().Len())
	for i := 0; i < enumDescriptor.Values().Len(); i++ {
		value := enumDescriptor.Values().Get(i)
		names[int32(value.Number())] = string(value.Name())
		values[string(value.Name())] = int32(value.Number())
		allowed[i] = string(value.Name())
	}
	return &protoEnumFlag[T]{
		names:   names,
		values:  values,
		Allowed: allowed,
	}
}

type protoEnumFlag[T ProtoEnum] struct {
	Allowed []string
	names   map[int32]string
	values  map[string]int32
	Value   T
}

func (a *protoEnumFlag[T]) String() string {
	return a.names[int32(a.Value)]
}

func (a *protoEnumFlag[T]) Set(p string) error {
	v, ok := a.values[p]
	if !ok {
		return fmt.Errorf("%s is not one of [%s]", p, strings.Join(a.Allowed, ","))
	}
	a.Value = T(v)
	return nil
}

func (a *protoEnumFlag[T]) Type() string {
	return "string"
}
