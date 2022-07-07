package go_eth_client

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var _ Client = (*EthRPC)(nil)

type EthRPC struct {
	url        string
	client     *http.Client
	log        *log.Logger
	Debug      bool
	privateKey *ecdsa.PrivateKey
	cid        *big.Int
}

// New create new rpc client with given url
func New(url string, configPath string) (*EthRPC, error) {
	var keyPath string
	keyPath = filepath.Join(configPath, "account.key")

	keyByte, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	var password string
	psdPath := filepath.Join(configPath, "password")
	psd, err := ioutil.ReadFile(psdPath)
	if err != nil {
		return nil, err
	}
	password = strings.TrimSpace(string(psd))

	unlockedKey, err := keystore.DecryptKey(keyByte, password)
	if err != nil {
		return nil, err
	}
	etherCli, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	Cid, err := etherCli.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	rpc := &EthRPC{
		url:        url,
		client:     http.DefaultClient,
		privateKey: unlockedKey.PrivateKey,
		cid:        Cid,
		log:        log.New(os.Stderr, "", log.LstdFlags),
	}
	return rpc, nil
}

func (e EthRPC) InvokeEthContract(abiPath, address string, method, args string) (common.Hash, error) {
	//TODO implement me
	panic("implement me")
}

func (e EthRPC) CompileContract(code string) (*CompileResult, error) {
	//TODO implement me
	panic("implement me")
}

func (e EthRPC) Compile(codePath string, local bool) (*CompileResult, error) {
	//TODO implement me
	panic("implement me")
}

func (e EthRPC) Deploy(url, codePath, argContract string, local bool) (string, *CompileResult, error) {
	//TODO implement me
	panic("implement me")
}
