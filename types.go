package go_eth_client

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type CompileResult struct {
	Abi   []string
	Bin   []string
	Names []string
}

type TransactionOptions struct {
	Gas      uint64
	GasPrice *big.Int
	Nonce    uint64
}

type Option func(opts *bind.TransactOpts)

func WithNonce(nonce *big.Int) Option {
	return func(opts *bind.TransactOpts) {
		opts.Nonce = nonce
	}
}

func WithGasPrice(price *big.Int) Option {
	return func(opts *bind.TransactOpts) {
		opts.GasPrice = price
	}
}

func WithGasLimit(limit uint64) Option {
	return func(opts *bind.TransactOpts) {
		opts.GasLimit = limit
	}
}

type TransactionOption func(opts *TransactionOptions)

func WithTxNonce(nonce uint64) TransactionOption {
	return func(opts *TransactionOptions) {
		opts.Nonce = nonce
	}
}

func WithTxGasPrice(price *big.Int) TransactionOption {
	return func(opts *TransactionOptions) {
		opts.GasPrice = price
	}
}

func WithTxGasLimit(limit uint64) TransactionOption {
	return func(opts *TransactionOptions) {
		opts.Gas = limit
	}
}
