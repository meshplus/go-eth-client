package common

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewAccount(accountPath, password, fileName string) (*ecdsa.PrivateKey, string, error) {
	account, err := keystore.StoreKey(accountPath, password, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return nil, "", err
	}
	privKeyFile := filepath.Join(accountPath, fileName)
	err = os.Rename(account.URL.Path, privKeyFile)
	if err != nil {
		panic(err)
	}
	return KeystoreToPrivateKey(privKeyFile, password)
}

func KeystoreToPrivateKey(privKeyFile, password string) (*ecdsa.PrivateKey, string, error) {
	keyjson, err := ioutil.ReadFile(privKeyFile)
	if err != nil {
		fmt.Println("read keyjson file failed：", err)
	}
	unlockedKey, err := keystore.DecryptKey(keyjson, password)
	if err != nil {

		return nil, "", err

	}
	privKey := unlockedKey.PrivateKey
	addr := crypto.PubkeyToAddress(unlockedKey.PrivateKey.PublicKey)
	return privKey, addr.String(), nil
}

func PrivateKeyToPublic(privateKey *ecdsa.PrivateKey) (*ecdsa.PublicKey, string, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, "", fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return publicKeyECDSA, address, nil
}
