package utils

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func Decode(Abi *abi.ABI, funcName string, args ...interface{}) ([]interface{}, error) {
	var method abi.Method
	var err error

	if funcName == "" {
		method = Abi.Constructor
	} else {
		method, err = getMethod(Abi, funcName)
		if err != nil {
			return nil, err
		}
	}

	if len(method.Inputs) > len(args) {
		return nil, fmt.Errorf("the num of inputs is %v, expectd %v", len(method.Inputs), len(args))
	}

	typedArgs := make([]interface{}, len(method.Inputs))
	for idx, input := range method.Inputs {
		typedArgs[idx], err = convert(input.Type, args[idx])
		if err != nil {
			return nil, fmt.Errorf("convert %s to %s failed: %s", args[idx], input.Type.String(), err.Error())
		}
	}
	return typedArgs, nil
}

func getMethod(ab *abi.ABI, method string) (abi.Method, error) {
	for k, v := range ab.Methods {
		if k == method {
			return v, nil
		}
	}
	return abi.Method{}, fmt.Errorf("method %s is not existed", method)
}

func UnpackOutput(abi *abi.ABI, method string, receipt string) ([]interface{}, error) {
	m, err := getMethod(abi, method)
	if err != nil {
		return nil, fmt.Errorf("get method %w", err)
	}
	if len(m.Outputs) == 0 {
		return nil, nil
	}
	receiptData := []byte(receipt)
	res, err := abi.Unpack(method, receiptData)
	if err != nil {
		return nil, fmt.Errorf("unpack result %w", err)
	}
	return res, nil
}

func convert(t abi.Type, input interface{}) (interface{}, error) {
	// array or slice
	switch t.T {
	case abi.ArrayTy:
		// make sure that the length of input equals to the t.Size
		var (
			fmtVal = make([]interface{}, t.Size)
			idx    int
		)
		reflectInput := reflect.ValueOf(input)
		switch reflectInput.Kind() {
		case reflect.String:
			if t.Size >= 1 {
				fmtVal[idx] = input
				idx++
			}
		case reflect.Slice:
			valLen := reflectInput.Len()
			var formatLen int
			if valLen < t.Size {
				formatLen = valLen
			} else {
				formatLen = t.Size
			}
			for idx = 0; idx < formatLen; idx++ {
				fmtVal[idx] = reflectInput.Index(idx).Interface()
			}
		}

		// complete input with default "" (empty string)
		for i := idx; i < t.Size; i++ {
			fmtVal[idx] = ""
		}
		// build the array (not slice)
		data := reflect.New(t.GetType()).Elem()
		for idx, val := range fmtVal {
			elem, err := convert(*t.Elem, val)
			if err != nil {
				return nil, err
			}
			data.Index(idx).Set(reflect.ValueOf(elem))
		}
		return data.Interface(), nil

	case abi.SliceTy:
		// todo: reflect
		var fmtVal []interface{}
		reflectInput := reflect.ValueOf(input)
		switch reflectInput.Kind() {
		case reflect.String:
			fmtVal = []interface{}{input}
		case reflect.Slice:
			inputLen := reflectInput.Len()
			fmtVal = make([]interface{}, inputLen)
			for i := 0; i < inputLen; i++ {
				fmtVal[i] = reflectInput.Index(i).Interface()
			}
		}

		data := reflect.MakeSlice(t.GetType(), len(fmtVal), len(fmtVal))
		for idx, val := range fmtVal {
			elem, err := convert(*t.Elem, val)
			if err != nil {
				return nil, err
			}
			data.Index(idx).Set(reflect.ValueOf(elem))
		}
		return data.Interface(), nil

	case abi.FixedBytesTy:
		if str, ok := input.(string); ok {
			return newFixedBytes(t.Size, str), nil
		}
	default:
		if str, ok := input.(string); ok {
			return newElement(t, str)
		}

	}
	return nil, fmt.Errorf("%s is not support", t.String())
}

// convert from string to basic type element
func newElement(t abi.Type, val string) (interface{}, error) {
	if t.T == abi.SliceTy || t.T == abi.ArrayTy {
		return nil, nil
	}
	var UNIT = 64
	var elem interface{}
	switch t.String() {
	case "uint8":
		num, err := strconv.ParseUint(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = uint8(num)
	case "uint16":
		num, err := strconv.ParseUint(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = uint16(num)
	case "uint32":
		num, err := strconv.ParseUint(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = uint32(num)
	case "uint64":
		num, err := strconv.ParseUint(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = num
	case "uint128", "uint256", "int128", "int256":
		var num *big.Int
		var ok bool
		if val == "" {
			num = big.NewInt(0)
		} else {
			num, ok = big.NewInt(0).SetString(val, 10)
			if !ok {
				return nil, fmt.Errorf("set big int failed")
			}
		}
		elem = num
	case "int8":
		num, err := strconv.ParseInt(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = int8(num)
	case "int16":
		num, err := strconv.ParseInt(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = int16(num)
	case "int32":
		num, err := strconv.ParseInt(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = int32(num)
	case "int64":
		num, err := strconv.ParseInt(val, 10, UNIT)
		if err != nil {
			return nil, err
		}
		elem = num
	case "bool":
		v, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		elem = v
	case "address":
		elem = common.HexToAddress(val)
	case "string":
		elem = val
	case "bytes":
		elem = common.Hex2Bytes(val)
	default:
		// default use reflect but do not use val
		// because it's impossible to know how to convert from string to target type
		elem = reflect.New(t.GetType()).Elem().Interface()
	}

	return elem, nil
}

var byteTy = reflect.TypeOf(byte(0))

// the return val is a byte array, not slice
func newFixedBytes(size int, val string) interface{} {
	// pre-define size 1,2,3...32 and 64, other size use reflect
	switch size {
	case 1:
		var data [1]byte
		copy(data[:], val)
		return data
	case 2:
		var data [2]byte
		copy(data[:], val)
		return data
	case 3:
		var data [3]byte
		copy(data[:], val)
		return data
	case 4:
		var data [4]byte
		copy(data[:], val)
		return data
	case 5:
		var data [5]byte
		copy(data[:], val)
		return data
	case 6:
		var data [6]byte
		copy(data[:], val)
		return data
	case 7:
		var data [7]byte
		copy(data[:], val)
		return data
	case 8:
		var data [8]byte
		copy(data[:], val)
		return data
	case 9:
		var data [9]byte
		copy(data[:], val)
		return data
	case 10:
		var data [10]byte
		copy(data[:], val)
		return data
	case 11:
		var data [11]byte
		copy(data[:], val)
		return data
	case 12:
		var data [12]byte
		copy(data[:], val)
		return data
	case 13:
		var data [13]byte
		copy(data[:], val)
		return data
	case 14:
		var data [14]byte
		copy(data[:], val)
		return data
	case 15:
		var data [15]byte
		copy(data[:], val)
		return data
	case 16:
		var data [16]byte
		copy(data[:], val)
		return data
	case 17:
		var data [17]byte
		copy(data[:], val)
		return data
	case 18:
		var data [18]byte
		copy(data[:], val)
		return data
	case 19:
		var data [19]byte
		copy(data[:], val)
		return data
	case 20:
		var data [20]byte
		copy(data[:], val)
		return data
	case 21:
		var data [21]byte
		copy(data[:], val)
		return data
	case 22:
		var data [22]byte
		copy(data[:], val)
		return data
	case 23:
		var data [23]byte
		copy(data[:], val)
		return data
	case 24:
		var data [24]byte
		copy(data[:], val)
		return data
	case 25:
		var data [25]byte
		copy(data[:], val)
		return data
	case 26:
		var data [26]byte
		copy(data[:], val)
		return data
	case 27:
		var data [27]byte
		copy(data[:], val)
		return data
	case 28:
		var data [28]byte
		copy(data[:], val)
		return data
	case 29:
		var data [29]byte
		copy(data[:], val)
		return data
	case 30:
		var data [30]byte
		copy(data[:], val)
		return data
	case 31:
		var data [31]byte
		copy(data[:], val)
		return data
	case 32:
		var data [32]byte
		copy(data[:], val)
		return data
	case 64:
		var data [64]byte
		copy(data[:], val)
		return data
	default:
		return newFixedBytesWithReflect(size, val)
	}
}

//! NOTICE: newFixedBytesWithReflect take more 15 times of time than newFixedBytes
//! So it is just use for those fixed bytes which are not commonly used.
func newFixedBytesWithReflect(size int, val string) interface{} {
	data := reflect.New(reflect.ArrayOf(size, byteTy)).Elem()
	bytes := reflect.ValueOf([]byte(val))
	reflect.Copy(data, bytes)
	return data.Interface()
}
