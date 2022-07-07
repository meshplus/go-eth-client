package go_eth_client

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/meshplus/bitxhub-model/pb"
	"math/big"
)

type Client interface {
	InvokeEthContract(abiPath, address string, method, args string) (common.Hash, error)
	Compile(codePath string, local bool) (*CompileResult, error)
	Deploy(url, codePath, argContract string, local bool) (string, *CompileResult, error)
	Call(method string, params ...interface{}) (json.RawMessage, error)
	EthGasPrice() (big.Int, error)
	EthGetTransactionReceipt(hash common.Hash) (*pb.Receipt, error)
	EthGetTransactionCount(address, block string) (int, error)
	EthSendTransaction(transaction *Transaction) (common.Hash, error)
	EthSendTransactionWithReceipt(transaction *Transaction) (*pb.Receipt, error)
	EthSendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	EthGetBalance(address, block string) (big.Int, error)
}
