package go_eth_client

import "math/big"

func (rpc *EthRPC) NewTransaction(nonce int, address string, amount big.Int, gas int, gasprice big.Int, packed string) Transaction {
	return Transaction{
		Nonce:    nonce,
		Gas:      gas,
		To:       address,
		Value:    amount,
		GasPrice: gasprice,
		payload:  packed,
	}
}
