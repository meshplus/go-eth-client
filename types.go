package go_eth_client

import (
	"crypto/ecdsa"
	"math/big"
)

type CompileResult struct {
	Abi   []string
	Bin   []string
	Names []string
}

type TransactionOptions struct {
	GasLimit   uint64
	GasPrice   *big.Int
	Nonce      uint64
	PrivateKey *ecdsa.PrivateKey
}

type Option func(opts *TransactionOptions)

func WithNonce(nonce uint64) Option {
	return func(opts *TransactionOptions) {
		opts.Nonce = nonce
	}
}

func WithGasPrice(price *big.Int) Option {
	return func(opts *TransactionOptions) {
		opts.GasPrice = price
	}
}

func WithGasLimit(limit uint64) Option {
	return func(opts *TransactionOptions) {
		opts.GasLimit = limit
	}
}
