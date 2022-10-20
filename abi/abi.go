package abi

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

type AbiEvent struct {
	Constructor *Struct
	InputMap    map[string]*Struct
	EventMap    map[string]*Struct
	OutputMap   map[string]*Struct
	Alias       map[string]string
	Abi         interface{}
}

func InitializeParameter(contractName, contractAddr string) (*AbiEvent, error) {
	contractPath := path.Join(h.contractPath, contractName)
	if _, ok := h.contractAlias[contractName]; !ok {
		h.contractAlias[contractName] = contractAddr
	}

	abiName := fmt.Sprintf("%s.abi", contractName)

	abiJSON, err := common.ReadFileAsString(filepath.Join(contractPath, abiName))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("contract %s initialize contract err", contractName))
	}
	fAbi, err := abi2.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("contract %s initialize contract err", contractName))
	}
	eventMap := make(map[string]*builder.Struct)
	alias := make(map[string]string)
	constructor := builder.NewBuilder()
	for _, input := range fAbi.Constructor.Inputs {
		constructor.AddField(abi2.ToCamelCase(input.Name), input.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
	}
	constructorStruct := constructor.Build()

	for _, event := range fAbi.Events {
		alias[event.ID.Hex()] = event.Name
		key := event.ID.Hex()
		val := builder.NewBuilder()
		// add event
		for _, input := range event.Inputs {
			if input.Indexed {
				val.AddField(abi2.ToCamelCase(input.Name), input.Type.GetType(), "")
			} else {
				val.AddField(abi2.ToCamelCase(input.Name), input.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
			}
		}
		eventMap[key] = val.Build()
	}
	inputMap := make(map[string]*builder.Struct)
	outputMap := make(map[string]*builder.Struct)
	for _, method := range fAbi.Methods {
		key := method.Name
		inputVal := builder.NewBuilder()
		outputVal := builder.NewBuilder()
		needBuild := true
		// init for inputs
		for _, input := range method.Inputs {
			if input.Type.GetType() == input.Type.TupleType {
				tupleStruct := builder.Tuple2Struct(&input.Type)
				if len(method.Inputs) == 1 {
					inputMap[key] = tupleStruct
					needBuild = false
				}
				inputVal.AddField(abi2.ToCamelCase(input.Name), reflect.TypeOf(tupleStruct.New().Interface()),
					reflect.StructTag(fmt.Sprintf("abi:\"%s\"", input.Name)))
			} else {
				inputVal.AddField(abi2.ToCamelCase(input.Name), input.Type.GetType(),
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
				return nil, crerrors.ErrNoOutputName
			}

			if output.Type.GetType() == output.Type.TupleType {
				tupleStruct := builder.Tuple2Struct(&output.Type)
				if len(method.Outputs) == 1 {
					outputMap[key] = tupleStruct
					needBuild = false
				}
				outputVal.AddField(abi2.ToCamelCase(output.Name), reflect.TypeOf(tupleStruct.New().Interface()), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", output.Name)))
			} else {
				outputVal.AddField(abi2.ToCamelCase(output.Name), output.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\"", output.Name)))
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
		Abi:         &fAbi,
	}, nil
}

func decode(instance *AbiEvent, method string, params []byte) ([]interface{}, error) {
	inputStruct, ok := instance.InputMap[method]
	if !ok {
		return nil, crerrors.ErrNoMethodFound
	}
	inputInstance := inputStruct.New()
	if err := json.Unmarshal(params, inputInstance.Addr()); err != nil {
		return nil, err
	}
	return inputInstance.Iterate(), nil
}
