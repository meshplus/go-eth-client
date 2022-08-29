package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	rpcx "github.com/meshplus/go-eth-client"
)

var nonce uint64

func main() {
	account, err := rpcx.LoadAccount("./batchEth/config")
	if err != nil {
		fmt.Println(err)
	}
	client, err := rpcx.New("http://localhost:8545", account.PrivateKey)
	if err != nil {
		fmt.Println(err)
	}
	latestNonce, err := client.EthGetTransactionCount(account.Address.String(), "latest")
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
