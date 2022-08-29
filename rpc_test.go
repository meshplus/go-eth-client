package go_eth_client

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	address, result, err := client.Deploy("testdata/storage.sol", "", true)
	require.Nil(t, err)
	require.NotEqual(t, "", address)
	require.NotNil(t, result)
}

func TestCompile(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol", true)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestInvokeEthContract(t *testing.T) {
	account, err := LoadAccount("testdata/config")
	require.Nil(t, err)
	client, err := New("http://localhost:8881", account.PrivateKey)
	require.Nil(t, err)
	address, result, err := client.Deploy("testdata/storage.sol", "", true)
	require.Nil(t, err)
	require.NotEqual(t, "", address)
	require.NotNil(t, result)
	_, err = client.InvokeEthContract("testdata/storage.abi", address, "store", "2", 0)
	require.Nil(t, err)
	time.Sleep(time.Second * 1)
	res, err := client.InvokeEthContract("testdata/storage.abi", address, "retrieve", "", 0)
	require.Nil(t, err)
	v, ok := res[0].(*big.Int)
	require.Equal(t, true, ok)
	require.Equal(t, "2", v.String())
}
