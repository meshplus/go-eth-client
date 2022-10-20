package go_eth_client

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	abi2 "github.com/meshplus/go-eth-client/abi"
)

type Struct struct {
	typ reflect.Type
	// <fieldName : 索引> // 用于通过字段名称，从Builder的[]reflect.StructField中获取reflect.StructField
	index map[string]int
}

func Tuple2Struct(in *abi.Type) *Struct {
	out := abi2.NewBuilder()
	for i, elem := range in.TupleElems {
		if elem.GetType() == elem.TupleType {
			tupleStruct := Tuple2Struct(elem)
			tmpStruct := *tupleStruct
			out.AddField(abi2.ToCamelCase(in.TupleRawNames[i]), reflect.TypeOf(tmpStruct.New().Interface()), reflect.StructTag(fmt.Sprintf(`abi:"%s"`, in.TupleRawNames[i])))
		} else {
			out.AddField(abi2.ToCamelCase(in.TupleRawNames[i]), elem.GetType(), reflect.StructTag(fmt.Sprintf(`abi:"%s"`, in.TupleRawNames[i])))
		}
	}
	return out.Build()
}
