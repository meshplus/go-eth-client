package abi

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
)

type Builder struct {
	// 用于存储属性字段
	fileId []reflect.StructField
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) IsNil() bool {
	return b.fileId == nil
}

func (b *Builder) AddField(field string, typ reflect.Type, tag reflect.StructTag) *Builder {
	b.fileId = append(b.fileId, reflect.StructField{Name: field, Type: typ, Tag: tag})
	return b
}

func (b *Builder) Build() *Struct {
	if b.fileId == nil {
		return &Struct{}
	}
	stu := reflect.StructOf(b.fileId)
	index := make(map[string]int)
	for i := 0; i < stu.NumField(); i++ {
		index[stu.Field(i).Name] = i
	}
	return &Struct{stu, index}
}

type Struct struct {
	typ reflect.Type
	// <fieldName : 索引> // 用于通过字段名称，从Builder的[]reflect.StructField中获取reflect.StructField
	index map[string]int
}

func (s *Struct) IsNil() bool {
	return s.index == nil
}

func (s *Struct) String() string {
	res := "Struct {\n"
	for name, index := range s.index {
		res += fmt.Sprintf("\t%-10s\t%-10s\t`%-10s`\n", name, s.typ.Field(index).Type, s.typ.Field(index).Tag)
	}
	res += "}"
	return res
}

func (s *Struct) New() *Instance {
	return &Instance{reflect.New(s.typ).Elem(), s.index}
}

type Instance struct {
	instance reflect.Value
	// <fieldName : 索引>
	index map[string]int
}

func (in *Instance) Iterate() []interface{} {
	var res []interface{}
	count := in.instance.NumField()
	for i := 0; i < count; i++ {
		res = append(res, in.instance.Field(i).Interface())
	}
	return res
}

func (in *Instance) SetField(name string, value interface{}) *Instance {
	// todo(lrx): if wrapper error is better?
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(errors.Errorf("set field err: %s", r))
		}
	}()
	in.instance.FieldByName(name).Set(reflect.ValueOf(value))
	return in
}

func (in *Instance) Interface() interface{} {
	return in.instance.Interface()
}

func (in *Instance) Addr() interface{} {
	return in.instance.Addr().Interface()
}

func (in *Instance) String() string {
	res := "Instance {\n"
	for name, index := range in.index {
		res += fmt.Sprintf("\t%-10s:%-10v\n", name, in.instance.Field(index).Interface())
	}
	res += "}"
	return res
}

// Indirect transfer a struct ptr back to struct, it is an implementation of json unmarshall
func Indirect(v reflect.Value, decodingNull bool) reflect.Value {
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if decodingNull && v.CanSet() {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return v
}

func Tuple2Struct(in *abi.Type) *Struct {
	out := NewBuilder()
	for i, elem := range in.TupleElems {
		if elem.GetType() == elem.TupleType {
			tupleStruct := Tuple2Struct(elem)
			tmpStruct := *tupleStruct
			out.AddField(abi.ToCamelCase(in.TupleRawNames[i]), reflect.TypeOf(tmpStruct.New().Interface()), reflect.StructTag(fmt.Sprintf(`abi:"%s"`, in.TupleRawNames[i])))
		} else {
			out.AddField(abi.ToCamelCase(in.TupleRawNames[i]), elem.GetType(), reflect.StructTag(fmt.Sprintf(`abi:"%s"`, in.TupleRawNames[i])))
		}
	}
	return out.Build()
}
