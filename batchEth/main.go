package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	rpcx "github.com/meshplus/go-eth-client"
)

var nonce uint64

func main() {
	client, err := rpcx.New("http://localhost:8545", "./batchEth/config")
	keyPath := filepath.Join("./batchEth/config", "account.key")

	keyByte, err := ioutil.ReadFile(keyPath)
	if err != nil {
		fmt.Println(err)
	}

	var password string
	psdPath := filepath.Join("./batchEth/config", "password")
	psd, err := ioutil.ReadFile(psdPath)
	if err != nil {
		fmt.Println(err)
	}
	password = strings.TrimSpace(string(psd))

	unlockedKey, err := keystore.DecryptKey(keyByte, password)
	if err != nil {
		fmt.Println(err)
	}
	latestNonce, err := client.EthGetTransactionCount(unlockedKey.Address.String(), "latest")
	if err != nil {
		fmt.Println(err)
	}
	nonce = uint64(latestNonce)
	var wg sync.WaitGroup

	count := 13
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			_, err = client.InvokeEthContract("./batchEth/config/data_swapper.abi", "0x3E98b263335ed03F9A6E1719dB1c61F158c14855", "get", "1356:appchain2:0xd9c987613Be7e38A7ef95aba5d3C4CF98791A3F5^bob1", atomic.AddUint64(&nonce, 1)-1)
			if err != nil {
				fmt.Println(err)
				return
			}
			wg.Done()
			fmt.Printf("success send tx%d\n", i)
		}(i)
	}
	wg.Wait()
	fmt.Println("success send tx count:", count)
}
