package go_eth_client

import (
	"encoding/json"
	"github.com/meshplus/bitxhub-model/pb"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Client interface {
	InvokeEthContract(abiPath, address string, method, args string) (*pb.Receipt, error)
	Compile(codePath string, local bool) (*CompileResult, error)
	Deploy(codePath, argContract string, local bool) (string, *CompileResult, error)
	Call(method string, params ...interface{}) (json.RawMessage, error)
	EthGasPrice() (big.Int, error)
	EthGetTransactionReceipt(hash common.Hash) (*pb.Receipt, error)
	EthGetTransactionCount(address, block string) (int, error)
	EthSendTransaction(transaction *Transaction) (common.Hash, error)
	EthSendTransactionWithReceipt(transaction *Transaction) (*pb.Receipt, error)
	EthSendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	EthGetBalance(address, block string) (big.Int, error)
}
