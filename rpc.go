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
	client *ethclient.Client
	cid    *big.Int
}

func New(url string) (*EthRPC, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	cid, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	return &EthRPC{
		client: client,
		cid:    cid,
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

func (rpc *EthRPC) Deploy(privateKey *ecdsa.PrivateKey, result *CompileResult, args []interface{}, opts ...Option) ([]string, error) {
	if len(result.Abi) == 0 || len(result.Bin) == 0 || len(result.Names) == 0 {
		return nil, fmt.Errorf("empty contract")
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, rpc.cid)
	if err != nil {
		return nil, err
	}
	//set transaction options
	for _, opt := range opts {
		opt(txOpts)
	}
	if txOpts.GasLimit == 0 {
		txOpts.GasLimit = 100000000
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
		address, _, _, err := bind.DeployContract(txOpts, parsed, common.FromHex(code), rpc.client, args...)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address.String())
	}
	return addresses, nil
}

func (rpc *EthRPC) Invoke(privateKey *ecdsa.PrivateKey, contractAbi *abi.ABI, address string, method string, args []interface{}, opts ...TransactionOption) ([]interface{}, error) {
	from := crypto.PubkeyToAddress(privateKey.PublicKey)
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
		txOpts := &TransactionOptions{}
		for _, opt := range opts {
			opt(txOpts)
		}
		if txOpts.Nonce == 0 {
			nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privateKey.PublicKey), nil)
			if err != nil {
				return nil, err
			}
			txOpts.Nonce = nonce
		}
		if txOpts.Gas == 0 {
			txOpts.Gas = 1000000
		}
		if txOpts.GasPrice == nil {
			price, err := rpc.EthGasPrice()
			if err != nil {
				return nil, err
			}
			txOpts.GasPrice = price
		}
		tx := NewTransaction(txOpts.Nonce, to, txOpts.Gas, txOpts.GasPrice, packed, nil)
		hash, err := rpc.EthSendTransaction(tx, privateKey)
		if err != nil {
			return nil, err
		}
		return []interface{}{hash.String()}, nil
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

func (rpc *EthRPC) EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balance, err := rpc.client.BalanceAt(context.Background(), account, blockNumber)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (rpc *EthRPC) EthSendTransaction(transaction *types.Transaction, privateKey *ecdsa.PrivateKey) (common.Hash, error) {
	signTx, err := types.SignTx(transaction, types.NewEIP155Signer(rpc.cid), privateKey)
	if err != nil {
		return common.Hash{}, err
	}
	err = rpc.client.SendTransaction(context.Background(), signTx)
	if err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), nil
}

func (rpc *EthRPC) EthSendTransactionWithReceipt(transaction *types.Transaction, privateKey *ecdsa.PrivateKey) (*types.Receipt, error) {
	hash, err := rpc.EthSendTransaction(transaction, privateKey)
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

func (rpc *EthRPC) Close() {
	rpc.client.Close()
}
