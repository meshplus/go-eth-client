package go_eth_client

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var _ Client = (*EthRPC)(nil)

type EthRPC struct {
	url        string
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	cid        *big.Int
}

func New(url string, pk *ecdsa.PrivateKey) (*EthRPC, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	cid, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	return &EthRPC{
		url:        url,
		client:     client,
		privateKey: pk,
		cid:        cid,
	}, nil
}

func (rpc *EthRPC) Compile(sourceFiles ...string) (*CompileResult, error) {
	contracts, err := compiler.CompileSolidity("", sourceFiles...)
	if err != nil {
		return nil, fmt.Errorf("compile contract: %w", err)
	}
	var abis, bins, names []string
	for name, contract := range contracts {
		contractAbi, err := json.Marshal(contract.Info.AbiDefinition)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABIs from compiler output: %w", err)
		}
		abis = append(abis, string(contractAbi))
		bins = append(bins, contract.Code)
		names = append(names, name)
	}
	return &CompileResult{
		Abi:   abis,
		Bin:   bins,
		Names: names,
	}, nil
}

func (rpc *EthRPC) DeployByCode(abi abi.ABI, code string, args []interface{}, opts ...Option) (string, uint64, error) {
	privKey := rpc.privateKey
	transactionOpts := &TransactionOptions{}
	//set transaction options
	for _, opt := range opts {
		opt(transactionOpts)
	}
	// load privateKey
	if transactionOpts.PrivateKey != nil {
		privKey = transactionOpts.PrivateKey
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, rpc.cid)
	if err != nil {
		return "", 0, err
	}
	txOpts.GasPrice = transactionOpts.GasPrice

	if transactionOpts.Nonce == 0 {
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privKey.PublicKey), nil)
		if err != nil {
			return "", 0, err
		}
		transactionOpts.Nonce = nonce
	}
	txOpts.Nonce = big.NewInt(int64(transactionOpts.Nonce))
	if transactionOpts.GasLimit == 0 {
		txOpts.GasLimit = 100000000
	} else {
		txOpts.GasLimit = transactionOpts.GasLimit
	}

	address, tx, _, err := bind.DeployContract(txOpts, abi, common.FromHex(code), rpc.client, args...)
	if err != nil {
		return "", 0, err
	}
	// try three times
	var receipt *types.Receipt
	if err := retry.Retry(func(attempt uint) error {
		receipt, err = rpc.client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			return err
		}
		return nil
	}, strategy.Wait(2*time.Second), strategy.Limit(5)); err != nil {
		return "", 0, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return "", 0, fmt.Errorf("deploy contract failed, tx hash is: %s", tx.Hash())
	}
	return address.String(), receipt.BlockNumber.Uint64(), nil
}

func (rpc *EthRPC) Deploy(result *CompileResult, args []interface{}, opts ...Option) ([]string, error) {
	if len(result.Abi) == 0 || len(result.Bin) == 0 || len(result.Names) == 0 {
		return nil, fmt.Errorf("empty contract")
	}

	privKey := rpc.privateKey
	transactionOpts := &TransactionOptions{}
	//set transaction options
	for _, opt := range opts {
		opt(transactionOpts)
	}
	// load privateKey
	if transactionOpts.PrivateKey != nil {
		privKey = transactionOpts.PrivateKey
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, rpc.cid)
	if err != nil {
		return nil, err
	}
	txOpts.GasPrice = transactionOpts.GasPrice

	if transactionOpts.Nonce == 0 {
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privKey.PublicKey), nil)
		if err != nil {
			return nil, err
		}
		transactionOpts.Nonce = nonce
	}

	txOpts.Nonce = big.NewInt(int64(transactionOpts.Nonce))
	if transactionOpts.GasLimit == 0 {
		txOpts.GasLimit = 100000000
	} else {
		txOpts.GasLimit = transactionOpts.GasLimit
	}

	addresses := make([]string, 0)
	for i, bin := range result.Bin {
		if bin == "0x" {
			continue
		}
		parsed, err := abi.JSON(strings.NewReader(result.Abi[i]))
		if err != nil {
			return nil, err
		}
		code := strings.TrimPrefix(strings.TrimSpace(bin), "0x")

		address, tx, _, err := bind.DeployContract(txOpts, parsed, common.FromHex(code), rpc.client, args...)
		if err != nil {
			return nil, err
		}
		// try three times
		var receipt *types.Receipt
		if err := retry.Retry(func(attempt uint) error {
			receipt, err = rpc.client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				return err
			}
			return nil
		}, strategy.Wait(1*time.Second), strategy.Limit(3)); err != nil {
			return nil, err
		}
		if receipt.Status == types.ReceiptStatusFailed {
			return nil, fmt.Errorf("deploy contract failed, tx hash is: %s", tx.Hash())
		}
		addresses = append(addresses, address.String())
	}
	return addresses, nil
}

func (rpc *EthRPC) Invoke(contractAbi *abi.ABI, address string, method string, args []interface{}, opts ...Option) ([]interface{}, error) {
	privkey := rpc.privateKey
	txOpts := &TransactionOptions{}
	for _, opt := range opts {
		opt(txOpts)
	}
	if txOpts.PrivateKey != nil {
		privkey = txOpts.PrivateKey
	}
	from := crypto.PubkeyToAddress(privkey.PublicKey)
	to := common.HexToAddress(address)
	packed, err := contractAbi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: from, To: &to, Data: packed}
	if contractAbi.Methods[method].IsConstant() {
		output, err := rpc.client.CallContract(context.Background(), msg, nil)
		if err != nil {
			return nil, err
		}
		if len(output) == 0 {
			if code, err := rpc.EthCodeAt(to, nil); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, fmt.Errorf("no code at your contract addresss")
			}
			return nil, fmt.Errorf("output is empty")
		}
		// unpack result for display
		result, err := UnpackOutput(contractAbi, method, string(output))
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		if txOpts.Nonce == 0 {
			nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privkey.PublicKey), nil)
			if err != nil {
				return nil, err
			}
			txOpts.Nonce = nonce
		}
		if txOpts.GasLimit == 0 {
			txOpts.GasLimit = 1000000
		}
		if txOpts.GasPrice == nil {
			price, err := rpc.EthGasPrice()
			if err != nil {
				return nil, err
			}
			txOpts.GasPrice = price
		}
		tx := NewTransaction(txOpts.Nonce, to, txOpts.GasLimit, txOpts.GasPrice, packed, nil)
		receipt, err := rpc.EthSendTransactionWithReceipt(tx, privkey)
		if err != nil {
			return nil, err
		}
		if receipt.Status == types.ReceiptStatusFailed {
			return nil, fmt.Errorf("invoke error: %v", receipt.TxHash.String())
		}
		return []interface{}{receipt.TxHash.String()}, nil
	}
}

func (rpc *EthRPC) EthGasPrice() (*big.Int, error) {
	price, err := rpc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	return price, nil
}

func (rpc *EthRPC) EthGetTransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	var err error
	err = retry.Retry(func(attempt uint) error {
		receipt, err = rpc.client.TransactionReceipt(context.Background(), hash)
		if err != nil {
			return err
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(500*time.Millisecond)))
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (rpc *EthRPC) EthGetTransactionCount(account common.Address, blockNumber *big.Int) (uint64, error) {
	nonce, err := rpc.client.NonceAt(context.Background(), account, blockNumber)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (rpc *EthRPC) EthGetBlockByNumber(blockNumber *big.Int, excludingTxs bool) (*types.Block, error) {
	var (
		err   error
		block *types.Block
	)
	if excludingTxs {
		blockHeder, err := rpc.client.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			return nil, err
		}
		return types.NewBlockWithHeader(blockHeder), nil
	}
	block, err = rpc.client.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (rpc *EthRPC) EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balance, err := rpc.client.BalanceAt(context.Background(), account, blockNumber)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (rpc *EthRPC) EthSendTransaction(transaction *types.Transaction, inputKey *ecdsa.PrivateKey) (common.Hash, error) {
	privKey := inputKey
	if inputKey == nil {
		if rpc.privateKey == nil {
			return [32]byte{}, fmt.Errorf("client privatekey is empty")
		}
		privKey = rpc.privateKey
	}

	signTx, err := types.SignTx(transaction, types.NewEIP155Signer(rpc.cid), privKey)
	if err != nil {
		return common.Hash{}, err
	}
	err = rpc.client.SendTransaction(context.Background(), signTx)
	if err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), nil
}

func (rpc *EthRPC) EthSendTransactionWithReceipt(transaction *types.Transaction, inputKey *ecdsa.PrivateKey) (*types.Receipt, error) {
	hash, err := rpc.EthSendTransaction(transaction, inputKey)
	if err != nil {
		return nil, err
	}

	receipt, err := rpc.EthGetTransactionReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (rpc *EthRPC) EthCodeAt(account common.Address, blockNumber *big.Int) ([]byte, error) {
	code, err := rpc.client.CodeAt(context.Background(), account, blockNumber)
	if err != nil {
		return nil, err
	}
	return code, nil
}

func (rpc *EthRPC) EthGetChainId() *big.Int {
	return rpc.cid
}
