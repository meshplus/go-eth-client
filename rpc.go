package go_eth_client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/Rican7/retry/backoff"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types1 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	CONTRACT = "contract_"
)

var _ Client = (*EthRPC)(nil)

type EthRPC struct {
	url        string
	client     *http.Client
	etherCli   *ethclient.Client
	log        *log.Logger
	Debug      bool
	privateKey *ecdsa.PrivateKey
	cid        *big.Int
}

// New create new rpc client with given url
func New(url string, configPath string) (*EthRPC, error) {
	var keyPath string
	keyPath = filepath.Join(configPath, "account.key")

	keyByte, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	var password string
	psdPath := filepath.Join(configPath, "password")
	psd, err := ioutil.ReadFile(psdPath)
	if err != nil {
		return nil, err
	}
	password = strings.TrimSpace(string(psd))

	unlockedKey, err := keystore.DecryptKey(keyByte, password)
	if err != nil {
		return nil, err
	}
	etherCli, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	Cid, err := etherCli.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	rpc := &EthRPC{
		url:        url,
		client:     http.DefaultClient,
		etherCli:   etherCli,
		privateKey: unlockedKey.PrivateKey,
		cid:        Cid,
		log:        log.New(os.Stderr, "", log.LstdFlags),
	}
	return rpc, nil
}

func (rpc EthRPC) InvokeEthContract(abiPath, address string, method, args string) ([]interface{}, error) {
	file, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return nil, err
	}
	ab, err := abi.JSON(bytes.NewReader(file))
	if err != nil {
		return nil, err
	}
	// prepare for invoke parameters
	var argx []interface{}
	if len(args) != 0 {
		argSplits := strings.Split(args, "^")
		var argArr []interface{}
		for _, arg := range argSplits {
			if strings.Index(arg, "[") == 0 && strings.LastIndex(arg, "]") == len(arg)-1 {
				if len(arg) == 2 {
					argArr = append(argArr, make([]string, 0))
					continue
				}
				// deal with slice
				argSp := strings.Split(arg[1:len(arg)-1], ",")
				argArr = append(argArr, argSp)
				continue
			}
			argArr = append(argArr, arg)
		}
		argx, err = Encode(ab, method, argArr...)
		if err != nil {
			return nil, err
		}
	}
	fromAddress := crypto.PubkeyToAddress(rpc.privateKey.PublicKey)
	toAddress := common.HexToAddress(address)
	packed, err := ab.Pack(method, argx...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: fromAddress, To: &toAddress, Data: packed}
	if ab.Methods[method].IsConstant() {
		output, err := rpc.etherCli.CallContract(context.Background(), msg, nil)
		if err != nil {
			return nil, err
		}
		if len(output) == 0 {
			if code, err := rpc.etherCli.CodeAt(context.Background(), toAddress, nil); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, fmt.Errorf("no code at your contract addresss")
			}
			return nil, fmt.Errorf("output is empty")
		}
		// unpack result for display
		result, err := UnpackOutput(ab, method, string(output))
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return result, nil
	} else {
		gasLimit := uint64(1000000)
		gasPrice, err := rpc.EthGasPrice()
		pubKey := rpc.privateKey.Public()
		publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		}
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(*publicKeyECDSA).String(), "latest")
		tx := types1.NewTx(&types1.LegacyTx{
			Nonce:    uint64(nonce),
			To:       &toAddress,
			Gas:      gasLimit,
			GasPrice: &gasPrice,
			Data:     packed,
		})
		signTx, err := types1.SignTx(tx, types1.NewEIP155Signer(big.NewInt(1356)), rpc.privateKey)
		if err != nil {
			return nil, err
		}
		data, err := signTx.MarshalBinary()
		rawTx := hexutil.Bytes(data)
		hash, err := rpc.EthSendRawTransaction(rawTx)
		if err != nil {
			return nil, err
		}
		var res []interface{}
		res = append(res, hash)
		return res, nil
	}
}

func (rpc EthRPC) InvokeEthContractByDefaultAbi(ab ethabi.ABI, address string, method, args string) ([]interface{}, error) {
	// prepare for invoke parameters
	var argx []interface{}
	var err error
	if len(args) != 0 {
		argSplits := strings.Split(args, "^")
		var argArr []interface{}
		for _, arg := range argSplits {
			if strings.Index(arg, "[") == 0 && strings.LastIndex(arg, "]") == len(arg)-1 {
				if len(arg) == 2 {
					argArr = append(argArr, make([]string, 0))
					continue
				}
				// deal with slice
				argSp := strings.Split(arg[1:len(arg)-1], ",")
				argArr = append(argArr, argSp)
				continue
			}
			argArr = append(argArr, arg)
		}
		fmt.Println("argArr", argArr)
		argx, err = Encode(ab, method, argArr...)
		if err != nil {
			return nil, err
		}
	}
	fromAddress := crypto.PubkeyToAddress(rpc.privateKey.PublicKey)
	toAddress := common.HexToAddress(address)
	packed, err := ab.Pack(method, argx...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: fromAddress, To: &toAddress, Data: packed}
	if ab.Methods[method].IsConstant() {
		output, err := rpc.etherCli.CallContract(context.Background(), msg, nil)
		if err != nil {
			return nil, err
		}
		if len(output) == 0 {
			if code, err := rpc.etherCli.CodeAt(context.Background(), toAddress, nil); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, fmt.Errorf("no code at your contract addresss")
			}
			return nil, fmt.Errorf("output is empty")
		}
		// unpack result for display
		result, err := UnpackOutput(ab, method, string(output))
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return result, nil
	} else {
		gasLimit := uint64(1000000)
		gasPrice, err := rpc.EthGasPrice()
		pubKey := rpc.privateKey.Public()
		publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		}
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(*publicKeyECDSA).String(), "latest")
		tx := types1.NewTx(&types1.LegacyTx{
			Nonce:    uint64(nonce),
			To:       &toAddress,
			Gas:      gasLimit,
			GasPrice: &gasPrice,
			Data:     packed,
		})
		signTx, err := types1.SignTx(tx, types1.NewEIP155Signer(big.NewInt(1356)), rpc.privateKey)
		if err != nil {
			return nil, err
		}
		data, err := signTx.MarshalBinary()
		rawTx := hexutil.Bytes(data)
		hash, err := rpc.EthSendRawTransaction(rawTx)
		if err != nil {
			return nil, err
		}
		var res []interface{}
		res = append(res, hash)
		return res, nil
	}
}

func (rpc EthRPC) compileContract(code string) (*CompileResult, error) {
	data, err := rpc.Call("contract_"+"compileContract", code)
	if err != nil {
		return nil, err
	}

	var cr CompileResult
	if sysErr := json.Unmarshal(data, &cr); sysErr != nil {
		return nil, sysErr
	}
	return &cr, nil
}

func (rpc EthRPC) Compile(codePath string, local bool) (*CompileResult, error) {
	if !local {
		return rpc.compileContract(codePath)
	}
	codePaths := strings.Split(codePath, ",")
	contracts, err := compiler.CompileSolidity("", codePaths...)
	if err != nil {
		return nil, fmt.Errorf("compile contract: %w", err)
	}

	var (
		abis  []string
		bins  []string
		types []string
	)
	for name, contract := range contracts {
		Abi, err := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABIs from compiler output: %w", err)
		}
		abis = append(abis, string(Abi))
		bins = append(bins, contract.Code)
		types = append(types, name)
	}

	result := &CompileResult{
		Abi:   abis,
		Bin:   bins,
		Types: types,
	}
	return result, nil
}

func (rpc EthRPC) Deploy(codePath, argContract string, local bool) (string, *CompileResult, error) {
	// compile solidity first
	compileResult, err := rpc.Compile(codePath, local)
	if err != nil {
		return "", nil, err
	}

	var addr common.Address

	if len(compileResult.Abi) == 0 || len(compileResult.Bin) == 0 || len(compileResult.Types) == 0 {
		return "", nil, fmt.Errorf("empty contract")
	}
	auth, err := bind.NewKeyedTransactorWithChainID(rpc.privateKey, rpc.cid)
	if err != nil {
		return "", nil, err
	}
	auth.GasLimit = 100000000
	for i, bin := range compileResult.Bin {
		if bin == "0x" {
			continue
		}
		parsed, err := abi.JSON(strings.NewReader(compileResult.Abi[i]))
		if err != nil {
			return "", nil, err
		}
		code := strings.TrimPrefix(strings.TrimSpace(bin), "0x")
		// prepare for constructor parameters
		var argx []interface{}
		if len(argContract) != 0 {
			argSplits := strings.Split(argContract, "^")
			var argArr []interface{}
			for _, arg := range argSplits {
				if strings.Index(arg, "[") == 0 && strings.LastIndex(arg, "]") == len(arg)-1 {
					if len(arg) == 2 {
						argArr = append(argArr, make([]string, 0))
						continue
					}
					// deal with slice
					argSp := strings.Split(arg[1:len(arg)-1], ",")
					argArr = append(argArr, argSp)
					continue
				}
				argArr = append(argArr, arg)
			}
			argx, err = Encode(parsed, "", argArr...)
			if err != nil {
				return "", nil, err
			}
		}
		addr1, tx, _, err := bind.DeployContract(auth, parsed, common.FromHex(code), rpc.etherCli, argx...)
		addr = addr1
		if err != nil {
			return "", nil, err
		}
		var r *types1.Receipt
		if err := retry.Retry(func(attempt uint) error {
			r, err = rpc.etherCli.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				return err
			}

			return nil
		}, strategy.Wait(1*time.Second)); err != nil {
			return "", nil, err
		}

		if r.Status == types1.ReceiptStatusFailed {
			return "", nil, fmt.Errorf("deploy contract failed, tx hash is: %s", r.TxHash.Hex())
		}
		//write abi file
		dir := filepath.Dir(compileResult.Types[i])
		base := filepath.Base(compileResult.Types[i])
		ext := filepath.Ext(compileResult.Types[i])
		f := strings.TrimSuffix(base, ext)
		filename := fmt.Sprintf("%s.abi", f)
		p := filepath.Join(dir, filename)
		err = ioutil.WriteFile(p, []byte(compileResult.Abi[i]), 0644)
		if err != nil {
			return "", nil, err
		}
	}
	return addr.Hex(), compileResult, nil
}

// Call returns raw response of method call
func (rpc *EthRPC) Call(method string, params ...interface{}) (json.RawMessage, error) {
	request := ethRequest{
		ID:      1,
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := rpc.client.Post(rpc.url, "application/json", bytes.NewBuffer(body))
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if rpc.Debug {
		rpc.log.Println(fmt.Sprintf("%s\nRequest: %s\nResponse: %s\n", method, body, data))
	}

	resp := new(ethResponse)
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, err
	}
	return resp.Result, nil

}

func (rpc *EthRPC) call(method string, target interface{}, params ...interface{}) error {
	result, err := rpc.Call(method, params...)
	if err != nil {
		return err
	}

	if target == nil {
		return nil
	}

	return json.Unmarshal(result, target)
}

// EthGasPrice returns the current price per gas in wei.
func (rpc *EthRPC) EthGasPrice() (big.Int, error) {
	var response string
	if err := rpc.call("eth_gasPrice", &response); err != nil {
		return big.Int{}, err
	}

	return ParseBigInt(response)
}

// EthGetTransactionReceipt returns the receipt of a transaction by transaction hash.
// Note That the receipt is not available for pending transactions.
func (rpc *EthRPC) EthGetTransactionReceipt(hash common.Hash) (*types1.Receipt, error) {
	Receipt := new(types1.Receipt)
	err := retry.Retry(func(attempt uint) error {
		err := rpc.call("eth_getTransactionReceipt", Receipt, hash)
		if err != nil {
			return err
		}
		return nil
	},
		strategy.Limit(5),
		strategy.Backoff(backoff.Fibonacci(500*time.Millisecond)),
	)
	if err != nil {
		return nil, err
	}
	return Receipt, nil
}

// EthGetTransactionCount returns the number of transactions sent from an address.
func (rpc *EthRPC) EthGetTransactionCount(address, block string) (int, error) {
	var response string

	if err := rpc.call("eth_getTransactionCount", &response, address, block); err != nil {
		return 0, err
	}

	return ParseInt(response)
}

// EthSendTransaction creates new message call transaction or a contract creation, if the data field contains code.
func (rpc *EthRPC) EthSendTransaction(transaction *Transaction) (common.Hash, error) {
	var hash common.Hash
	too := common.HexToAddress(transaction.To)
	pubkey := rpc.privateKey.Public()
	publicKeyECDSA, ok := pubkey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(*publicKeyECDSA).String(), "latest")
	if err != nil {
		return hash, err
	}
	gasLimit := uint64(21000)
	gasPrice, err := rpc.EthGasPrice()
	tx := types1.NewTx(&types1.LegacyTx{
		Nonce:    uint64(nonce),
		To:       &too,
		Value:    transaction.Value,
		Gas:      gasLimit,
		GasPrice: &gasPrice,
		Data:     []byte{},
	})
	signTx, err := types1.SignTx(tx, types1.NewEIP155Signer(big.NewInt(1356)), rpc.privateKey)
	if err != nil {
		return hash, err
	}
	data, err := signTx.MarshalBinary()
	rawTx := hexutil.Bytes(data)
	return rpc.EthSendRawTransaction(rawTx)
}

func (rpc *EthRPC) EthSendTransactionWithReceipt(transaction *Transaction) (*types1.Receipt, error) {
	too := common.HexToAddress(transaction.To)
	pubkey := rpc.privateKey.Public()
	publicKeyECDSA, ok := pubkey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(*publicKeyECDSA).String(), "latest")
	if err != nil {
		return nil, err
	}
	gasLimit := uint64(21000)
	gasPrice, err := rpc.EthGasPrice()
	tx := types1.NewTx(&types1.LegacyTx{
		Nonce:    uint64(nonce),
		To:       &too,
		Value:    transaction.Value,
		Gas:      gasLimit,
		GasPrice: &gasPrice,
		Data:     []byte{},
	})
	signTx, err := types1.SignTx(tx, types1.NewEIP155Signer(big.NewInt(1356)), rpc.privateKey)
	if err != nil {
		return nil, err
	}
	data, err := signTx.MarshalBinary()
	rawTx := hexutil.Bytes(data)
	hash, err := rpc.EthSendRawTransaction(rawTx)
	if err != nil {
		return nil, err
	}
	return rpc.EthGetTransactionReceipt(hash)
}

// EthSendRawTransaction creates new message call transaction or a contract creation for signed transactions.
func (rpc *EthRPC) EthSendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	var hash common.Hash
	err := rpc.call("eth_sendRawTransaction", &hash, data)
	return hash, err
}

// EthGetBalance returns the balance of the account of given address in wei.
func (rpc *EthRPC) EthGetBalance(address, block string) (big.Int, error) {
	var response string
	if err := rpc.call("eth_getBalance", &response, address, block); err != nil {
		return big.Int{}, err
	}
	return ParseBigInt(response)
}

func (rpc *EthRPC) InvokeContract(method string, params ...interface{}) (*types1.Receipt, error) {
	var receipt types1.Receipt
	err := rpc.call(method, &receipt, params)
	return &receipt, err
}

func (rpc EthRPC) Invoke(ab ethabi.ABI, address string, method string, args []interface{}) ([]interface{}, error) {
	// prepare for invoke parameters
	var err error
	fromAddress := crypto.PubkeyToAddress(rpc.privateKey.PublicKey)
	toAddress := common.HexToAddress(address)
	packed, err := ab.Pack(method, args)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: fromAddress, To: &toAddress, Data: packed}
	if ab.Methods[method].IsConstant() {
		output, err := rpc.etherCli.CallContract(context.Background(), msg, nil)
		if err != nil {
			return nil, err
		}
		if len(output) == 0 {
			if code, err := rpc.etherCli.CodeAt(context.Background(), toAddress, nil); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, fmt.Errorf("no code at your contract addresss")
			}
			return nil, fmt.Errorf("output is empty")
		}
		// unpack result for display
		result, err := UnpackOutput(ab, method, string(output))
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return result, nil
	} else {
		gasLimit := uint64(1000000)
		gasPrice, err := rpc.EthGasPrice()
		pubKey := rpc.privateKey.Public()
		publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		}
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(*publicKeyECDSA).String(), "latest")
		tx := types1.NewTx(&types1.LegacyTx{
			Nonce:    uint64(nonce),
			To:       &toAddress,
			Gas:      gasLimit,
			GasPrice: &gasPrice,
			Data:     packed,
		})
		signTx, err := types1.SignTx(tx, types1.NewEIP155Signer(big.NewInt(1356)), rpc.privateKey)
		if err != nil {
			return nil, err
		}
		data, err := signTx.MarshalBinary()
		rawTx := hexutil.Bytes(data)
		hash, err := rpc.EthSendRawTransaction(rawTx)
		if err != nil {
			return nil, err
		}
		var res []interface{}
		res = append(res, hash)
		return res, nil
	}
}
