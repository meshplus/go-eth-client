package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	pe := NewBuilder().
		AddField("Name", reflect.TypeOf(""), `abi:"name"`).
		AddField("Age", reflect.TypeOf(int64(0)), `abi:"age"`)
	assert.False(t, pe.IsNil())
	peStruct := pe.Build()
	assert.False(t, peStruct.IsNil())
	assert.NotEqual(t, "", peStruct.String())
	p := peStruct.New()
	assert.NotNil(t, p.SetField("Name", "你好"))
	assert.NotNil(t, p.SetField("Age", int64(32)))
	assert.Nil(t, p.SetField("Age", 32))
	assert.Equal(t, p.Iterate(), []interface{}{"你好", int64(32)})
	assert.NotEqual(t, "", p.String())

	fmt.Printf("%T，%+v\n", p.Interface(), p.Interface())
	fmt.Printf("%T，%+v\n", p.Addr(), p.Addr())
	data, err := json.Marshal(p.Interface())
	assert.Nil(t, err)
	fmt.Println(data)

	test := &struct {
		Name string
		Age  int64
	}{}
	err = json.Unmarshal(data, test)
	assert.Nil(t, err)
	fmt.Println(test)
}

func TestInterfaceConvert(t *testing.T) {
	inputs := []interface{}{"Alice", big.NewInt(10)}

	b := NewBuilder().
		AddField("Key", reflect.TypeOf(""), `abi:"key"`).
		AddField("Balance", reflect.TypeOf(big.NewInt(0)), `abi:"balance"`)

	bStruct := b.Build()
	ins := bStruct.New()

	ins.BatchSetFields(inputs)

	type BalanceRes struct {
		Key     string   `abi:"key"`
		Balance *big.Int `abi:"balance"`
	}
	balanceRes := &BalanceRes{}
	data, err := json.Marshal(ins.Interface())
	assert.Nil(t, err)
	err = json.Unmarshal(data, balanceRes)
	assert.Nil(t, err)
	assert.Equal(t, balanceRes.Key, "Alice")
	assert.Equal(t, balanceRes.Balance, big.NewInt(10))
}

func TestIndict(t *testing.T) {
	type unmarshalStruct struct {
		A string
		B int64
	}
	res := &unmarshalStruct{
		A: "Alice",
		B: 10,
	}
	re := reflect.ValueOf(res)
	v := Indirect(re, false)
	assert.Equal(t, res.A, v.FieldByName("A").String())
	assert.Equal(t, reflect.TypeOf(unmarshalStruct{}), v.Type())
}

func TestEmbedded(t *testing.T) {
	type Em struct {
		Id    *big.Int
		Value string
	}
	emInstance := Em{Id: big.NewInt(111), Value: "test-111"}

	b := NewBuilder().AddField("Name", reflect.TypeOf(""), "").
		AddField("Em", reflect.TypeOf(emInstance), "")
	bStruct := b.Build()
	bInstance := bStruct.New()
	bInstance.SetField("Name", "liu").SetField("Em", emInstance)

	type test struct {
		Name string
		Em   Em
	}
	data, err := json.Marshal(bInstance.Interface())
	assert.Nil(t, err)

	var res test
	err = json.Unmarshal(data, &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Em.Id, big.NewInt(111))
	assert.Equal(t, res.Em.Value, "test-111")

	// test embed struct
	data, err = json.Marshal(bInstance.Interface())
	assert.Nil(t, err)

	err = json.Unmarshal(data, &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Em.Id, big.NewInt(111))
	assert.Equal(t, res.Em.Value, "test-111")

	emInstance1 := Em{
		Id:    big.NewInt(1),
		Value: "ttt",
	}
	b1 := NewBuilder().AddField("Name", reflect.TypeOf(""), "").
		AddField("Em", reflect.TypeOf(emInstance1), "")
	bInstance1 := b1.Build().New()
	data, err = json.Marshal(res)
	assert.Nil(t, err)
	fmt.Println("marshal data", string(data))
	err = json.Unmarshal(data, bInstance1.Addr())
	assert.Nil(t, err)
	fmt.Println("ins1", bInstance1.instance.FieldByName("Em"))

	// test embed utils struct
	emStruct := NewBuilder().AddField("Id", reflect.TypeOf(&big.Int{}), "").
		AddField("Value", reflect.TypeOf(""), "").Build()

	emInstance2 := emStruct.New().SetField("Id", big.NewInt(111)).
		SetField("Value", "test-111")
	b2 := NewBuilder().AddField("Name", reflect.TypeOf(""), "").
		AddField("Em", reflect.TypeOf(emInstance2.Interface()), "")
	bStruct2 := b2.Build()
	bInstance2 := bStruct2.New()
	// bInstance2.SetField("Em", emInstance2).SetField("Name", "liu")

	res.Em.Id = big.NewInt(222)
	res.Em.Value = "test-222"

	data, err = json.Marshal(res)
	assert.Nil(t, err)
	fmt.Println("marshal data112", string(data))
	// fmt.Println("old", bInstance2.String())
	err = json.Unmarshal(data, bInstance2.Addr())
	assert.Nil(t, err)
	fmt.Println("ins2", bInstance2)
}

func TestTuple2Struct(t *testing.T) {
	t.Run("testSimpleTuple", func(t *testing.T) {
		const simpleTuple = `[{"name":"testTuple","type":"function","outputs":[{"type":"tuple","name":"ret","components":[{"type":"int256","name":"a"},{"type":"int256","name":"b"}]}]}]`
		abi, err := abi.JSON(strings.NewReader(simpleTuple))
		if err != nil {
			t.Fatal(err)
		}
		buff := new(bytes.Buffer)

		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // ret[a] = 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000011")) // ret[b] = 17

		type Output struct {
			A *big.Int
			B *big.Int
		}
		var expected = Output{A: big.NewInt(1), B: big.NewInt(17)}
		var res Output

		// test unpack value to struct
		err = abi.UnpackIntoInterface(&res, "testTuple", buff.Bytes())
		assert.Nil(t, err)
		reflect.DeepEqual(res, expected)

		// test unpack value to utils instance
		outputMap, err := initialParam(abi)
		assert.Nil(t, err)
		instance := outputMap["testTuple"].New()
		err = abi.UnpackIntoInterface(instance.Addr(), "testTuple", buff.Bytes())
		assert.Nil(t, err)
		reflect.DeepEqual(instance.Interface(), expected)

		// test instance to struct
		data, err := json.Marshal(instance.Interface())
		var res1 Output
		err = json.Unmarshal(data, &res1)
		assert.Nil(t, err)
		reflect.DeepEqual(res1, expected)
	})

	t.Run("testNestedTuple", func(t *testing.T) {
		// test nested tuple
		const nestedTuple = `[{"name":"testTuple","type":"function","outputs":[
		{"type":"tuple","name":"s","components":[{"type":"uint256","name":"a"},{"type":"uint256[]","name":"b"},{"type":"tuple[]","name":"c","components":[{"name":"x", "type":"uint256"},{"name":"y","type":"uint256"}]}]},
		{"type":"tuple","name":"t","components":[{"name":"x", "type":"uint256"},{"name":"y","type":"uint256"}]},
		{"type":"uint256","name":"a"}
	]}]`

		abi, err := abi.JSON(strings.NewReader(nestedTuple))
		if err != nil {
			t.Fatal(err)
		}
		buff := new(bytes.Buffer)
		buff.Reset()
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // s offset
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")) // t.X = 0
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // t.Y = 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // a = 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.A = 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // s.B offset
		buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000c0")) // s.C offset
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.B length
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.B[0] = 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.B[0] = 2
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C length
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.C[0].X
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C[0].Y
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C[1].X
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.C[1].Y

		type T struct {
			X *big.Int `abi:"x"`
			Z *big.Int `abi:"y"` // test whether the abi tag works.
		}

		type S struct {
			A *big.Int
			B []*big.Int
			C []T
		}

		type Ret struct {
			S S `abi:"s"`
			T T `abi:"t"`
			A *big.Int
		}
		var ret Ret
		var expected = Ret{
			S: S{
				A: big.NewInt(1),
				B: []*big.Int{big.NewInt(1), big.NewInt(2)},
				C: []T{
					{big.NewInt(1), big.NewInt(2)},
					{big.NewInt(2), big.NewInt(1)},
				},
			},
			T: T{
				big.NewInt(0), big.NewInt(1),
			},
			A: big.NewInt(1),
		}

		// test unpack value to struct
		err = abi.UnpackIntoInterface(&ret, "testTuple", buff.Bytes())
		if err != nil {
			t.Error(err)
		}
		if reflect.DeepEqual(ret, expected) {
			t.Error("unexpected unpack value")
		}

		// test unpack value to utils instance
		outputMap, err := initialParam(abi)
		assert.Nil(t, err)
		instance := outputMap["testTuple"].New()
		err = abi.UnpackIntoInterface(instance.Addr(), "testTuple", buff.Bytes())
		assert.Nil(t, err)
		reflect.DeepEqual(instance.Interface(), expected)

		// test instance to struct
		data, err := json.Marshal(instance.Interface())
		var res1 Ret
		err = json.Unmarshal(data, &res1)
		assert.Nil(t, err)
		reflect.DeepEqual(res1, expected)
	})
}

func TestCopy(t *testing.T) {
	const nestedTuple = `[{"name":"testTuple","type":"function","outputs":[
		{"type":"tuple","name":"s","components":[{"type":"uint256","name":"a"},{"type":"uint256[]","name":"b"},{"type":"tuple[]","name":"c","components":[{"name":"x", "type":"uint256"},{"name":"y","type":"uint256"}]}]},
		{"type":"tuple","name":"t","components":[{"name":"x", "type":"uint256"},{"name":"y","type":"uint256"}]},
		{"type":"uint256","name":"a"}
	]}]`

	bxhAbi, err := abi.JSON(strings.NewReader(nestedTuple))
	if err != nil {
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // s offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")) // t.X = 0
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // t.Y = 1
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // a = 1
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.A = 1
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // s.B offset
	buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000c0")) // s.C offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.B length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.B[0] = 1
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.B[0] = 2
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.C[0].X
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C[0].Y
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // s.C[1].X
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // s.C[1].Y

	type T struct {
		X *big.Int `abi:"x"`
		Z *big.Int `abi:"y"` // test whether the abi tag works.
	}

	type S struct {
		A *big.Int
		B []*big.Int
		C []T
	}

	type Ret struct {
		S S `abi:"s"`
		T T `abi:"t"`
		A *big.Int
	}
	var expected = Ret{
		S: S{
			A: big.NewInt(1),
			B: []*big.Int{big.NewInt(1), big.NewInt(2)},
			C: []T{
				{big.NewInt(1), big.NewInt(2)},
				{big.NewInt(2), big.NewInt(1)},
			},
		},
		T: T{
			big.NewInt(0), big.NewInt(1),
		},
		A: big.NewInt(1),
	}
	unpackData, err := bxhAbi.Unpack("testTuple", buff.Bytes())
	assert.Nil(t, err)

	hpcAbi, err := abi.JSON(strings.NewReader(nestedTuple))
	if err != nil {
		t.Fatal(err)
	}
	outputMap, err := initialParam(hpcAbi)
	assert.Nil(t, err)
	instance := outputMap["testTuple"].New()

	arguments, err := GetArguments("testTuple", &bxhAbi)
	assert.Nil(t, err)
	err = Copy(instance.Addr(), unpackData, arguments)
	assert.Nil(t, err)

	// test instance to struct
	data, err := json.Marshal(instance.Interface())
	var res Ret
	err = json.Unmarshal(data, &res)
	assert.Nil(t, err)
	reflect.DeepEqual(res, expected)
}

func initialParam(abi abi.ABI) (map[string]*Struct, error) {
	inputVal := NewBuilder()
	outputVal := NewBuilder()
	outputMap := make(map[string]*Struct)
	flag := false
	for _, method := range abi.Methods {
		key := method.Name
		for _, input := range method.Inputs {
			inputVal.AddField(strings.ToTitle(input.Name), input.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\" json:\"%s\"", input.Name, input.Name)))
		}
		for _, output := range method.Outputs {
			if output.Name == "" {
				return nil, fmt.Errorf("empty abi output Name")
			}
			if output.Type.GetType() == output.Type.TupleType {
				tupleStruct := Tuple2Struct(&output.Type)
				if len(method.Outputs) == 1 {
					outputMap[key] = tupleStruct
					flag = true
				}
				outputVal.AddField(strings.ToTitle(output.Name), reflect.TypeOf(tupleStruct.New().Interface()), reflect.StructTag(fmt.Sprintf("abi:\"%s\" json:\"%s\"", output.Name, output.Name)))
			} else {
				outputVal.AddField(strings.ToTitle(output.Name), output.Type.GetType(), reflect.StructTag(fmt.Sprintf("abi:\"%s\" json:\"%s\"", output.Name, output.Name)))
			}
		}
		if !flag {
			outputMap[key] = outputVal.Build()
		}
	}
	return outputMap, nil
}
