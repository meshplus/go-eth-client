package utils

import (
	"bytes"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func LoadAbi(abiPath string) (abi.ABI, error) {
	file, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return abi.ABI{}, err
	}
	contractAbi, err := abi.JSON(bytes.NewReader(file))
	if err != nil {
		return abi.ABI{}, err
	}
	return contractAbi, nil
}

func NewTransaction(nonce uint64, address common.Address, gas uint64, gasPrice *big.Int, data []byte, value *big.Int) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &address,
		Gas:      gas,
		GasPrice: gasPrice,
		Data:     data,
		Value:    value,
	})
}

func NewDynamicFeeTransaction(chainid *big.Int, nonce uint64, address common.Address, gas uint64, gasPrice *big.Int, data []byte, value *big.Int) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainid,
		Nonce:     nonce,
		GasTipCap: gasPrice,
		GasFeeCap: gasPrice,
		Gas:       gas,
		To:        &address,
		Data:      data,
		Value:     value,
	})
}
