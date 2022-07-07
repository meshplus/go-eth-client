package go_eth_client

import (
	"github.com/ethereum/go-ethereum/common"
)

type Client interface {
	InvokeEthContract(abiPath, address string, method, args string) (common.Hash, error)
	CompileContract(code string) (*CompileResult, error)
	Compile(codePath string, local bool) (*CompileResult, error)
	Deploy(url, codePath, argContract string, local bool) (string, *CompileResult, error)
}
