package utils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type AbiEvent struct {
	Constructor *Struct
	InputMap    map[string]*Struct
	EventMap    map[string]*Struct
	OutputMap   map[string]*Struct
	Alias       map[string]string
	Abi         interface{}
}

func DecodeBytes(instance *AbiEvent, method string, params []byte) ([]interface{}, error) {
	inputStruct, ok := instance.InputMap[method]
	if !ok {
		return nil, fmt.Errorf("method not found")
	}
	inputInstance := inputStruct.New()
	if err := json.Unmarshal(params, inputInstance.Addr()); err != nil {
		return nil, err
	}
	return inputInstance.Iterate(), nil
}

func InitializeParameter(contractPath string) (*AbiEvent, error) {
	contractAbi, err := LoadAbi(contractPath)
	if err != nil {
		return nil, err
	}
	eventMap := make(map[string]*Struct)
	alias := make(map[string]string)
	constructor := NewBuilder()
	for _, input := range contractAbi.Constructor.Inputs {
		constructor.AddField(abi.ToCamelCase(input.Name), input.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
	}
	constructorStruct := constructor.Build()

	for _, event := range contractAbi.Events {
		alias[event.ID.Hex()] = event.Name
		key := event.ID.Hex()
		val := NewBuilder()
		// add event
		for _, input := range event.Inputs {
			if input.Indexed {
				val.AddField(abi.ToCamelCase(input.Name), input.Type.GetType(), "")
			} else {
				val.AddField(abi.ToCamelCase(input.Name), input.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
			}
		}
		eventMap[key] = val.Build()
	}
	inputMap := make(map[string]*Struct)
	outputMap := make(map[string]*Struct)
	for _, method := range contractAbi.Methods {
		key := method.Name
		inputVal := NewBuilder()
		outputVal := NewBuilder()
		needBuild := true
		// init for inputs
		for _, input := range method.Inputs {
			if input.Type.GetType() == input.Type.TupleType {
				tupleStruct := Tuple2Struct(&input.Type)
				if len(method.Inputs) == 1 {
					inputMap[key] = tupleStruct
					needBuild = false
				}
				inputVal.AddField(abi.ToCamelCase(input.Name), reflect.TypeOf(tupleStruct.New().Interface()),
					reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
			} else {
				inputVal.AddField(abi.ToCamelCase(input.Name), input.Type.GetType(),
					reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
			}
		}
		if needBuild {
			inputMap[key] = inputVal.Build()
		}

		// reset flag for output
		needBuild = true
		for _, output := range method.Outputs {
			if output.Name == "" {
				return nil, fmt.Errorf("empty abi output name")
			}

			if output.Type.GetType() == output.Type.TupleType {
				tupleStruct := Tuple2Struct(&output.Type)
				if len(method.Outputs) == 1 {
					outputMap[key] = tupleStruct
					needBuild = false
				}
				outputVal.AddField(abi.ToCamelCase(output.Name), reflect.TypeOf(tupleStruct.New().Interface()), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", output.Name)))
			} else {
				outputVal.AddField(abi.ToCamelCase(output.Name), output.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", output.Name)))
			}
		}
		if needBuild {
			outputMap[key] = outputVal.Build()
		}
	}
	return &AbiEvent{
		Constructor: constructorStruct,
		InputMap:    inputMap,
		EventMap:    eventMap,
		OutputMap:   outputMap,
		Alias:       alias,
		Abi:         &contractAbi,
	}, nil
}
