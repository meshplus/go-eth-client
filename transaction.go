package go_eth_client

import (
	"math/big"
)

func (rpc *EthRPC) NewTransaction(nonce int, address string, amount *big.Int, gas int, gasPrice big.Int, packed string) *Transaction {
	return &Transaction{
		Nonce:    nonce,
		Gas:      gas,
		To:       address,
		Value:    amount,
		GasPrice: gasPrice,
		payload:  packed,
	}
}
