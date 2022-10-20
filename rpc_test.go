package go_eth_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol")
	require.Nil(t, err)
	require.NotNil(t, result)
	fmt.Println(result)
}

func TestDeployByCode(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	file, err := ioutil.ReadFile("testdata/data.abi")
	assert.Nil(t, err)
	abi, err := abi.JSON(bytes.NewReader(file))
	require.Nil(t, err)
	code, err := ioutil.ReadFile("testdata/data.bin")
	require.Nil(t, err)
	address, err := client.DeployByCode(abi, string(code), nil)
	require.Nil(t, err)

	fmt.Println("register org")
	contractAbi, err := LoadAbi("testdata/data.abi")
	require.Nil(t, err)
	type signStruct struct {
		HashedMessage [32]byte
		V             uint8
		R             [32]byte
		S             [32]byte
	}
	//sign := new(signStruct)
	//signByte, err := json.Marshal(sign)
	args, err := Encode(contractAbi, "registerOrg", big.NewInt(123).String(), "test-123", "", "")
	require.Nil(t, err)
	_, err = client.Invoke(contractAbi, address, "registerOrg", args)
	require.Nil(t, err)

	fmt.Println("register 1st user")
	type userStruct struct {
		UserAddr string
		OrgId    *big.Int
		Credit   *big.Int
		Extra    string
	}
	type userInput struct {
		User userStruct
		Sign signStruct
	}
	userInstance := userInput{
		User: userStruct{
			UserAddr: account.Address.String(),
			OrgId:    big.NewInt(123),
			Credit:   big.NewInt(1000),
			Extra:    "",
		},
	}
	bodyBytes, err := json.Marshal(userInstance)
	assert.Nil(t, err)
	args, err = Encode(contractAbi, "registerUser", bodyBytes)
	require.Nil(t, err)
	_, err = client.Invoke(contractAbi, address, "registerUser", args)
	require.Nil(t, err)

	fmt.Println("register 2nd user")
	userInstance = userInput{
		User: userStruct{
			UserAddr: "0x47bd692d7728dee508a2791701d54597cc1b8100",
			OrgId:    big.NewInt(123),
			Credit:   big.NewInt(1000),
			Extra:    "",
		},
	}
	bodyBytes, err = json.Marshal(userInstance)
	assert.Nil(t, err)
	args, err = Encode(contractAbi, "registerUser", bodyBytes)
	require.Nil(t, err)
	_, err = client.Invoke(contractAbi, address, "registerUser", args)
	require.Nil(t, err)

	fmt.Println("publish data")
	type creditPackage struct {
		Credit   *big.Int
		Quantity uint8
		Duration *big.Int
	}

	type publishStruct struct {
		DataId      *big.Int
		Publisher   string
		Prices      []creditPackage
		AuthList    []*big.Int
		SharingMode uint8
		Extra       []byte
		Sign        signStruct
	}

	input := publishStruct{
		DataId:    big.NewInt(123),
		Publisher: account.Address.String(),
		Prices: []creditPackage{{
			Credit:   big.NewInt(10),
			Quantity: 0,
			Duration: big.NewInt(1000),
		}},
		AuthList:    nil,
		SharingMode: 0,
		Extra:       nil,
	}

	bodyBytes, err = json.Marshal(input)
	assert.Nil(t, err)

	args, err = Encode(contractAbi, "publish", bodyBytes)
	require.Nil(t, err)
	_, err = client.Invoke(contractAbi, address, "publish", args)
	require.Nil(t, err)
}

func TestDeploy(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol")
	require.Nil(t, err)
	addresses, err := client.Deploy(result, nil)
	require.Nil(t, err)
	require.Equal(t, 1, len(addresses))
}

func TestInvokeEthContract(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol")
	require.Nil(t, err)
	addresses, err := client.Deploy(result, nil)
	require.Nil(t, err)
	require.Equal(t, 1, len(addresses))

	time.Sleep(time.Second)
	contractAbi, err := LoadAbi("testdata/storage.abi")
	require.Nil(t, err)
	args, err := Encode(contractAbi, "store", "2")
	require.Nil(t, err)
	_, err = client.Invoke(contractAbi, addresses[0], "store", args)
	require.Nil(t, err)

	time.Sleep(time.Second)
	res, err := client.Invoke(contractAbi, addresses[0], "retrieve", nil)
	require.Nil(t, err)
	v, ok := res[0].(*big.Int)
	require.Equal(t, true, ok)
	require.Equal(t, "2", v.String())
}

func TestEthGasPrice(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	price, err := client.EthGasPrice()
	require.Nil(t, err)
	require.Equal(t, "50000", price.String())
}

func TestEthGetTransactionReceipt(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	nonce, err := client.EthGetTransactionCount(account.Address, nil)
	require.Nil(t, err)
	price, err := client.EthGasPrice()
	require.Nil(t, err)
	pk, err := crypto.GenerateKey()
	require.Nil(t, err)
	tx := NewTransaction(nonce, crypto.PubkeyToAddress(pk.PublicKey), uint64(10000000), price, nil, big.NewInt(1))
	hash, err := client.EthSendTransaction(tx)
	require.Nil(t, err)
	receipt, err := client.EthGetTransactionReceipt(hash)
	require.Nil(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

func TestEthGetTransactionCount(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	nonce, err := client.EthGetTransactionCount(account.Address, nil)
	require.Nil(t, err)
	require.NotNil(t, nonce)
}

func TestEthGetBalance(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	balance, err := client.EthGetBalance(account.Address, nil)
	require.Nil(t, err)
	require.NotNil(t, balance)
}

func TestEthSendTransaction(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	nonce, err := client.EthGetTransactionCount(account.Address, nil)
	require.Nil(t, err)
	price, err := client.EthGasPrice()
	require.Nil(t, err)
	pk, err := crypto.GenerateKey()
	require.Nil(t, err)
	tx := NewTransaction(nonce, crypto.PubkeyToAddress(pk.PublicKey), uint64(10000000), price, nil, big.NewInt(1))
	hash, err := client.EthSendTransaction(tx)
	require.Nil(t, err)
	require.NotNil(t, hash)
}

func TestEthSendTransactionWithReceipt(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	nonce, err := client.EthGetTransactionCount(account.Address, nil)
	require.Nil(t, err)
	price, err := client.EthGasPrice()
	require.Nil(t, err)
	pk, err := crypto.GenerateKey()
	require.Nil(t, err)
	tx := NewTransaction(nonce, crypto.PubkeyToAddress(pk.PublicKey), uint64(10000000), price, nil, big.NewInt(1))
	receipt, err := client.EthSendTransactionWithReceipt(tx)
	require.Nil(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

func TestEthCodeAt(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol")
	require.Nil(t, err)
	addresses, err := client.Deploy(result, nil)
	require.Nil(t, err)
	require.Equal(t, 1, len(addresses))
	code, err := client.EthCodeAt(common.HexToAddress(addresses[0]), nil)
	require.Nil(t, err)
	require.NotNil(t, code)
}
