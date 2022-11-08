package go_eth_client

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Client interface {
	Compile(sourceFiles ...string) (*CompileResult, error)
	Deploy(privKey *ecdsa.PrivateKey, result *CompileResult, args []interface{}, opts ...Option) ([]string, error)
	DeployByCode(privKey *ecdsa.PrivateKey, abi abi.ABI, code string, args []interface{}, opts ...Option) (string, uint64, error)
	Invoke(privKey *ecdsa.PrivateKey, contractAbi *abi.ABI, address string, method string, args []interface{}, opts ...Option) ([]interface{}, error)
	EthCall(contractAbi *abi.ABI, address string, method string, args []interface{}) ([]interface{}, error)
	EthGasPrice() (*big.Int, error)
	EthGetTransactionReceipt(hash common.Hash) (*types.Receipt, error)
	EthGetTransactionCount(account common.Address, blockNumber *big.Int) (uint64, error)
	EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error)
	EthSendTransaction(privKey *ecdsa.PrivateKey, transaction *types.Transaction) (common.Hash, error)
	EthSendRawTransaction(transaction *types.Transaction) (common.Hash, error)
	EthSendTransactionWithReceipt(privKey *ecdsa.PrivateKey, transaction *types.Transaction) (*types.Receipt, error)
	EthSendRawTransactionWithReceipt(transaction *types.Transaction) (*types.Receipt, error)
	EthGetCode(account common.Address, blockNumber *big.Int) (string, error)
	EthGetBlockByNumber(blockNumber *big.Int, fullTx bool) (*types.Block, error)
	EthGetChainId() *big.Int
	EthGetBlockTransactionCountByHash(hash common.Hash) (uint64, error)
	EthGetBlockTransactionCountByNumber(blockNumber *big.Int) (uint64, error)
	EthGetTransactionByHash(txHash common.Hash) (*types.Transaction, error)
	EthGetTransactionByBlockHashAndIndex(blockHash common.Hash, index int) (*types.Transaction, error)
	EthGetTransactionByBlockNumberAndIndex(blockNumber *big.Int, index int) (*types.Transaction, error)
	EthEstimateGas(msg ethereum.CallMsg) (uint64, error)
}
