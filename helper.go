package go_eth_client

import (
	"bytes"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func LoadAccount(configPath string) (*keystore.Key, error) {
	keyPath := filepath.Join(configPath, "account.key")
	keyByte, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	psdPath := filepath.Join(configPath, "password")
	psd, err := ioutil.ReadFile(psdPath)
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(string(psd))
	unlockedKey, err := keystore.DecryptKey(keyByte, password)
	if err != nil {
		return nil, err
	}
	return unlockedKey, nil
}

func LoadAbi(abiPath string) (*abi.ABI, error) {
	file, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return nil, err
	}
	contractAbi, err := abi.JSON(bytes.NewReader(file))
	if err != nil {
		return nil, err
	}
	return &contractAbi, nil
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
