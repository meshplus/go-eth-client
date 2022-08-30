package go_eth_client

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Client interface {
	Compile(sourceFiles ...string) (*CompileResult, error)
	Deploy(result *CompileResult, args []interface{}, opts ...Option) ([]string, error)
	Invoke(contractAbi *abi.ABI, address string, method string, args []interface{}, opts ...TransactionOption) ([]interface{}, error)
	EthGasPrice() (*big.Int, error)
	EthGetTransactionReceipt(hash common.Hash) (*types.Receipt, error)
	EthGetTransactionCount(account common.Address, blockNumber *big.Int) (uint64, error)
	EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error)
	EthSendTransaction(transaction *types.Transaction) (common.Hash, error)
	EthSendTransactionWithReceipt(transaction *types.Transaction) (*types.Receipt, error)
	EthCodeAt(account common.Address, blockNumber *big.Int) ([]byte, error)
}
