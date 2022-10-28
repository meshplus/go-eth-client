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
	address, blockNum, err := client.DeployByCode(abi, string(code), nil)
	require.Nil(t, err)
	latestNum, err := client.EthGetBlockByNumber(nil, true)
	require.Nil(t, err)
	require.Equal(t, blockNum, latestNum)

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

func TestGetLatestBlock(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	block, err := client.EthGetBlockByNumber(nil, false)
	require.Nil(t, err)
	for _, tx := range block.Transactions() {
		fmt.Println(tx.Type())
		fmt.Println(tx.To())
	}
	fmt.Println(block.Number().String())
}

func TestName(t *testing.T) {
	type test struct {
		A *big.Int
		B *big.Int
	}
	a := &test{}
	bytes, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var b struct{ A *big.Int }
	err = json.Unmarshal(bytes, &b)
	assert.Nil(t, err)
	fmt.Println(b)
}

func TestMarshal(t *testing.T) {
	normalData := "{\"difficulty\":\"0x0\",\"extraData\":\"\",\"gasLimit\":\"0x5f5e100\",\"gasUsed\":\"0x3e8\",\"hash\":\"0x0914e0d8b4D7895D10f3928E7f558fd32AdBac7B2B087384b0e9Cb259F66Ec28\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"miner\":\"0x0000000000000000000000000000000000000000\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"number\":\"0x67\",\"parentHash\":\"0xCBd4376aDC2d57525199c90dE12159cf402Cb0dd868C3ACb17B394A7c846B6d6\",\"receiptsRoot\":\"0xAe5F7EB53582804d7D5aC7E777D423544b9a13D07aA40C440E7970c134C21cF1\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"size\":\"0x2e3\",\"stateRoot\":\"0x21a4908c1Ee0fA65BBB2Ca797Ee19577B89F9456231f1Eff619765B28b412d9B\",\"timestamp\":\"0x17219447f5d5da40\",\"totalDifficulty\":\"0x0\",\"transactions\":[{\"blockHash\":\"0x0914e0d8b4d7895d10f3928e7f558fd32adbac7b2b087384b0e9cb259f66ec28\",\"blockNumber\":\"0x67\",\"from\":\"0xf9deeec58b690d89eba38efe579b1c946549e66a\",\"gas\":\"0x0\",\"gasPrice\":\"0x0\",\"hash\":\"0x1900e79f84397b9ef84683393477278639b4bdc6bc1f1e2b105c61b179968219\",\"input\":\"0x1201301801\",\"nonce\":\"0x3\",\"to\":\"0x60427f3ee6dea954b2365acc8243b4f458fa94ea\",\"transactionIndex\":\"0x0\",\"value\":\"0x0\",\"type\":\"0x0\",\"v\":null,\"r\":null,\"s\":null}],\"transactionsRoot\":\"0xc7f6868c1FF6F06B0812097C181304e47D22D1AD09a5F10C67148C67156EEf7d\",\"uncles\":[]}"
	evmData := "{\n  \"difficulty\": \"0x0\",\n  \"extraData\": \"\",\n  \"gasLimit\": \"0x5f5e100\",\n  \"gasUsed\": \"0x3e8\",\n  \"hash\": \"0x3A09F6b5aD5155c2FBC8BA3F4E7a24f79baae403Dc80DF8174826aD8bbb78C4d\",\n  \"logsBloom\": \"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\n  \"miner\": \"0x0000000000000000000000000000000000000000\",\n  \"mixHash\": \"0x0000000000000000000000000000000000000000000000000000000000000000\",\n  \"nonce\": \"0x0000000000000000\",\n  \"number\": \"0x42\",\n  \"parentHash\": \"0x535a0dA838A3aECe30d11699751F377de5566b7B61153390A625D26109224150\",\n  \"receiptsRoot\": \"0xDDa7543647313122731B19bB0f46d68010316Bc5Ae2f1Ae2202D09C9f69d5A5c\",\n  \"sha3Uncles\": \"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\n  \"size\": \"0x383\",\n  \"stateRoot\": \"0x691740b895580710db1cbF7dE70127e5363063D6eEB3fa3229a09c203B7BAe0A\",\n  \"timestamp\": \"0x1721935ada2cb950\",\n  \"totalDifficulty\": \"0x0\",\n  \"transactions\": [\n    {\n      \"blockHash\": \"0x3a09f6b5ad5155c2fbc8ba3f4e7a24f79baae403dc80df8174826ad8bbb78c4d\",\n      \"blockNumber\": \"0x42\",\n      \"from\": \"0x20f7fac801c5fc3f7e20cfbadaa1cdb33d818fa3\",\n      \"gas\": \"0x5f5e100\",\n      \"gasPrice\": \"0xc350\",\n      \"hash\": \"0xcd7cee6261b683370b0978ba56e01f986880ec40e195fea8d3531d2cdf0c4b3e\",\n      \"input\": \"0x6080604052348015600f57600080fd5b5060ac8061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80632e64cec11460375780636057361d14604c575b600080fd5b60005460405190815260200160405180910390f35b605c6057366004605e565b600055565b005b600060208284031215606f57600080fd5b503591905056fea264697066735822122095c7b2d81e556f6e9046dd127dfb9bb733120e6c76cfc102af02e6f56b83c19264736f6c634300080f0033\",\n      \"nonce\": \"0x38\",\n      \"to\": null,\n      \"transactionIndex\": \"0x0\",\n      \"value\": \"0x0\",\n      \"type\": \"0x0\",\n      \"v\": \"0xabb\",\n      \"r\": \"0x9b5bac7a274f0b2aca19f08d15e8f46344253842a559ab28100f77091eff6f1\",\n      \"s\": \"0x3b8c9db5874d2b73a21b502a82d0dbc7419f3a1e9667d706a74a498f33d66696\"\n    }\n  ],\n  \"transactionsRoot\": \"0xf013829d6e3676467a9902497CBFEb432D9f41CE10474eb80925D56D11cDd055\",\n  \"uncles\": []\n}"

	type txExtraInfo struct {
		BlockNumber *string         `json:"blockNumber,omitempty"`
		BlockHash   *common.Hash    `json:"blockHash,omitempty"`
		From        *common.Address `json:"from,omitempty"`
	}

	type rpcTransaction struct {
		tx *types.Transaction
		txExtraInfo
	}
	type rpcBlock struct {
		Hash         common.Hash      `json:"hash"`
		Transactions []rpcTransaction `json:"transactions"`
		UncleHashes  []common.Hash    `json:"uncles"`
	}

	var body rpcBlock
	err := json.Unmarshal([]byte(evmData), &body)
	assert.Nil(t, err)
	fmt.Println(body)

	var body1 rpcBlock
	data := []byte(normalData)
	fmt.Println(len(data))
	err = json.Unmarshal([]byte(normalData), &body1)
	assert.Nil(t, err)
	fmt.Println(body1)
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
	hash, err := client.EthSendTransaction(tx, client.privateKey)
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
	hash, err := client.EthSendTransaction(tx, client.privateKey)
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
	receipt, err := client.EthSendTransactionWithReceipt(tx, client.privateKey)
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
