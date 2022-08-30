package go_eth_client

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
