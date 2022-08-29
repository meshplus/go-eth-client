package go_eth_client

import (
	"encoding/json"
	"math/big"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types1 "github.com/ethereum/go-ethereum/core/types"
)

type Client interface {
	InvokeEthContract(abiPath, address string, method, args string, nonce uint64) ([]interface{}, error)
	Compile(codePath string, local bool) (*CompileResult, error)
	Deploy(codePath, argContract string, local bool) (string, *CompileResult, error)
	Call(method string, params ...interface{}) (json.RawMessage, error)
	EthGasPrice() (big.Int, error)
	EthGetTransactionReceipt(hash common.Hash) (*types1.Receipt, error)
	EthGetTransactionCount(address, block string) (int, error)
	EthSendTransaction(transaction *Transaction) (common.Hash, error)
	EthSendTransactionWithReceipt(transaction *Transaction) (*types1.Receipt, error)
	EthSendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	EthGetBalance(address, block string) (big.Int, error)
	Invoke(ab ethabi.ABI, address string, method string, args ...interface{}) ([]interface{}, error)
}
