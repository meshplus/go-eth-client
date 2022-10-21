package go_eth_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
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

func TestDeployNft(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/pandaNft.sol")
	require.Nil(t, err)
	parm := make([]interface{}, 2)
	parm[0] = "Kung Fu Panda"
	parm[1] = "KFP"
	abi1 := make([]string, 0)
	bin := make([]string, 0)
	name := make([]string, 0)
	for i, value := range result.Names {
		if result.Names[i] == "testdata/pandaNft.sol:PandaNft" {
			name = append(name, value)
			abi1 = append(abi1, result.Abi[i])
			bin = append(bin, result.Bin[i])
		}
	}
	compileResult := CompileResult{
		Abi:   abi1,
		Bin:   bin,
		Names: name,
	}
	addresses, err := client.Deploy(&compileResult, parm)
	fmt.Println(addresses)
	contractAbi, err := abi.JSON(bytes.NewReader([]byte(compileResult.Abi[0])))

	args, err := Encode(&contractAbi, "setBaseURI", "Kung Fu Panda #")
	require.Nil(t, err)
	_, err = client.Invoke(&contractAbi, addresses[0], "setBaseURI", args)
	require.Nil(t, err)

	/*args, err = Encode(&contractAbi, "mint", "0xdfaCcdb4b2d27adB6eb043f8074033c55a2ab0cc", "1")
	require.Nil(t, err)
	_, err = client.Invoke(&contractAbi, addresses[0], "mint", args)
	require.Nil(t, err)*/

	/*args, err := Encode(&contractAbi, "getBaseURI")
	require.Nil(t, err)
	_, err = client.Invoke(&contractAbi, addresses[0], "getBaseURI", args)
	require.Nil(t, err)

	args, err = Encode(&contractAbi, "tokenURI", "1")
	require.Nil(t, err)
	res, err := client.Invoke(&contractAbi, addresses[0], "tokenURI", args)
	require.Nil(t, err)
	fmt.Println(res)

	args, err = Encode(&contractAbi, "ownerOf", "1")
	require.Nil(t, err)
	res, err = client.Invoke(&contractAbi, addresses[0], "ownerOf", args)
	require.Nil(t, err)
	fmt.Println(res)

	args, err = Encode(&contractAbi, "tokensOfOwnerIn", "0xdfaCcdb4b2d27adB6eb043f8074033c55a2ab0cc")
	require.Nil(t, err)
	res, err = client.Invoke(&contractAbi, addresses[0], "tokensOfOwnerIn", args)
	require.Nil(t, err)
	fmt.Println(res)*/
}

func TestDeployErc20(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol")
	require.Nil(t, err)
	parm := make([]interface{}, 2)
	parm[0] = "BXH"
	parm[1] = "Token"

	abi2 := make([]string, 0)
	bin := make([]string, 0)
	name := make([]string, 0)
	for i, value := range result.Names {
		if result.Names[i] == "testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol:ERC20PresetMinterPauser" {
			name = append(name, value)
			abi2 = append(abi2, result.Abi[i])
			bin = append(bin, result.Bin[i])
		}
	}
	compileResult := CompileResult{
		Abi:   abi2,
		Bin:   bin,
		Names: name,
	}
	addresses, err := client.Deploy(&compileResult, parm)
	fmt.Println(addresses)
}

func TestGetBalance(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	num, err := client.EthGetBalance(common.HexToAddress("0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"), nil)
	require.Nil(t, err)
	fmt.Println(num)
}

func TestErc20Get(t *testing.T) {

	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://127.0.0.1:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol")
	require.Nil(t, err)
	parm := make([]interface{}, 2)
	parm[0] = "BXH"
	parm[1] = "Token"

	abi2 := make([]string, 0)
	bin := make([]string, 0)
	name := make([]string, 0)
	for i, value := range result.Names {
		if result.Names[i] == "testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol:ERC20PresetMinterPauser" {
			name = append(name, value)
			abi2 = append(abi2, result.Abi[i])
			bin = append(bin, result.Bin[i])
		}
	}
	compileResult := CompileResult{
		Abi:   abi2,
		Bin:   bin,
		Names: name,
	}
	addresses, err := client.Deploy(&compileResult, parm)

	fmt.Println(addresses)
	require.Nil(t, err)
	contractAbi, err := abi.JSON(bytes.NewReader([]byte(compileResult.Abi[0])))

	args, err := Encode(&contractAbi, "mint", "0x20f7fac801c5fc3f7e20cfbadaa1cdb33d818fa3", "100000000000000000000")
	require.Nil(t, err)
	_, err = client.Invoke(&contractAbi, addresses[0], "mint", args)
	require.Nil(t, err)

	time.Sleep(time.Second)
	args, err = Encode(&contractAbi, "balanceOf", "0x20f7fac801c5fc3f7e20cfbadaa1cdb33d818fa3")
	require.Nil(t, err)
	res, err := client.Invoke(&contractAbi, addresses[0], "balanceOf", args)
	fmt.Println(res)

	args, err = Encode(&contractAbi, "transfer", "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013", "1000000000000000000")
	res, err = client.Invoke(&contractAbi, addresses[0], "transfer", args)
	require.Nil(t, err)

	args, err = Encode(&contractAbi, "balanceOf", "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013")
	require.Nil(t, err)
	res, err = client.Invoke(&contractAbi, addresses[0], "balanceOf", args)
	fmt.Println(res)

	fmt.Println(addresses[0])
}

func TestSign(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://127.0.0.1:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol")
	require.Nil(t, err)
	parm := make([]interface{}, 2)
	parm[0] = "BXH"
	parm[1] = "Token"

	abi2 := make([]string, 0)
	bin := make([]string, 0)
	name := make([]string, 0)
	for i, value := range result.Names {
		if result.Names[i] == "testdata/contracts/token/ERC20/ERC20PresetMinterPauser.sol:ERC20PresetMinterPauser" {
			name = append(name, value)
			abi2 = append(abi2, result.Abi[i])
			bin = append(bin, result.Bin[i])
		}
	}
	compileResult := CompileResult{
		Abi:   abi2,
		Bin:   bin,
		Names: name,
	}
	addresses, err := client.Deploy(&compileResult, parm)
	fmt.Println(addresses)

	contractAbi, err := abi.JSON(bytes.NewReader([]byte(compileResult.Abi[0])))

	args, err := Encode(&contractAbi, "transfer", "0xAb8483F64d9C6d1EcF9b849Ae677dD3315835cb2", "1000000000000000000")
	packed, err := contractAbi.Pack("transfer", args...)
	nonce, err := client.EthGetTransactionCount(account.Address, nil)
	a, _ := client.EthGasPrice()
	fmt.Println(a)
	tx1 := NewTransaction(nonce, common.HexToAddress(addresses[0]), 100000, big.NewInt(50000), packed, nil)
	res, _ := json.Marshal(tx1)
	fmt.Println(string(res))
	contractAddr := common.HexToAddress(addresses[0])
	gas, err := client.EthEstimateGas(ethereum.CallMsg{
		From:     account.Address,
		To:       &contractAddr,
		Gas:      100000,
		GasPrice: big.NewInt(50000),
		Data:     packed,
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(gas)

	/*signTx, err := types.SignTx(tx1, types.NewEIP155Signer(client.cid), account.PrivateKey)
	res2, _ := json.Marshal(signTx)
	fmt.Println(string(res2))*/

}

func TestEstimate(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	fmt.Println(account.Address)
	num, err := client.EthGetBalance(account.Address, nil)
	require.Nil(t, err)
	fmt.Println(num)
	to := common.HexToAddress("0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013")
	a := 10000.15
	b := math.BigPow(10, 20).Int64()
	fmt.Println(a, b)
	value := big.NewInt(0)
	value.SetString("1000000000000000000000000", 10)

	gas, err := client.EthEstimateGas(ethereum.CallMsg{From: account.Address, To: &to, Value: value})

	require.Nil(t, err)
	fmt.Println(gas)

	/*	contractAbi, err := LoadAbi("testdata/erc20.abi")
		a := 0.01
		value := strconv.FormatInt(int64(float64(math.BigPow(10, 18).Int64())*a), 10)
		fmt.Println(value)
		args, err := Encode(contractAbi, "transfer", "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013", "200")
		to = common.HexToAddress("0xAEE28338495D64f242ACF563990f8ec684dF8Ce9")
		packed, err := contractAbi.Pack("transfer", args...)
		gas, err = client.EthEstimateGas(ethereum.CallMsg{From: account.Address, To: &to, Data: packed})
		require.Nil(t, err)
		fmt.Println(gas)*/
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

func TestEthCodeAt2(t *testing.T) {
	b := "insufficient funds for gas"
	a := "insufficient funds for xx"
	fmt.Println(strings.Contains(b, a))

}
