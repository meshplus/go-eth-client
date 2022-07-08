package go_eth_client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	client, err := New("http://localhost:8881", "testdata/config")
	require.Nil(t, err)
	address, result, err := client.Deploy("testdata/storage.sol", "", true)
	require.Nil(t, err)
	require.NotEqual(t, "", address)
	require.NotNil(t, result)
}

func TestCompile(t *testing.T) {
	client, err := New("http://localhost:8881", "testdata/config")
	require.Nil(t, err)
	result, err := client.Compile("testdata/storage.sol", true)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestInvokeEthContract(t *testing.T) {
	client, err := New("http://localhost:8881", "testdata/config")
	require.Nil(t, err)
	address, result, err := client.Deploy("testdata/storage.sol", "", true)
	require.Nil(t, err)
	require.NotEqual(t, "", address)
	require.NotNil(t, result)
	//TODO:send write tx err
	res, err := client.InvokeEthContract("testdata/storage.abi", address, "store", "2")
	require.Nil(t, err)
	fmt.Println(res)
	res, err = client.InvokeEthContract("testdata/storage.abi", address, "retrieve", "")
	require.Nil(t, err)
	fmt.Println(string(res))
}
