package go_eth_client

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var urls = []string{
	"http://localhost:8881",
	"http://localhost:8882",
	"http://localhost:8883",
	"http://localhost:8884",
}

func newClient() (*ethclient.Client, string, error) {
	randIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(urls))
	client, err := ethclient.Dial(urls[randIndex])
	if err != nil {
		return nil, "", err
	}
	return client, urls[randIndex], nil
}

func TestNewPool(t *testing.T) {
	_, err := NewPool(newClient, 4, 8, 1*time.Hour)
	require.Nil(t, err)
	_, err = NewPool(newClient, 0, 0, 0)
	require.Nil(t, err)
	_, err = NewPool(newClient, 20, 0, 0)
	require.Nil(t, err)
}

func TestPool_Close(t *testing.T) {
	pool, err := NewPool(newClient, 4, 8, 1*time.Hour)
	require.Nil(t, err)
	pool.Close()
	// close repeatedly
	pool.Close()
}

func TestPool_Get(t *testing.T) {
	pool, err := NewPool(newClient, 1, 1, 100*time.Millisecond)
	require.Nil(t, err)

	// normal
	client, err := pool.Get(context.Background())
	require.Nil(t, err)
	err = pool.Put(client)
	require.Nil(t, err)

	// idle timeout
	time.Sleep(100 * time.Millisecond)
	client, err = pool.Get(context.Background())
	require.Nil(t, err)

	// get from channel timeout
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Millisecond)
	_, err = pool.Get(ctx)
	require.Equal(t, fmt.Errorf("get client from pool timed out"), err)

	// get from closed pool
	pool.Close()
	_, err = pool.Get(context.Background())
	require.Equal(t, fmt.Errorf("pool is closed"), err)
}

func TestPool_Put(t *testing.T) {
	pool, err := NewPool(newClient, 1, 2, 100*time.Millisecond)
	require.Nil(t, err)

	// normal
	client, err := pool.Get(context.Background())
	require.Nil(t, err)
	err = pool.Put(client)
	require.Nil(t, err)

	// put into a full pool
	err = pool.Put(client)
	require.Equal(t, fmt.Errorf("put a client into a full pool"), err)

	// put into a closed pool
	pool.Close()
	err = pool.Put(client)
	require.Equal(t, fmt.Errorf("pool is closed"), err)
}
