package go_eth_client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	defaultInit        = 1
	defaultCap         = 1
	defaultIdleTimeout = 6 * time.Minute
)

// Factory is a function type creating an eth client
type Factory func() (*ethclient.Client, error)

// Pool is the eth client pool
type Pool struct {
	clients     chan *clientConn
	factory     Factory
	idleTimeout time.Duration
	mu          sync.RWMutex
}

// clientConn is the wrapper for an eth client conn
type clientConn struct {
	conn     *ethclient.Client
	timeUsed time.Time
}

// NewPool creates a new clients pool with the given initial amd maximum capacity,
// and the timeout for the idle clients.
func NewPool(factory Factory, init, capacity int, idleTimeout time.Duration) (*Pool, error) {
	if capacity <= 0 {
		capacity = defaultCap
	}
	if init < 0 {
		init = defaultInit
	}
	if init > capacity {
		init = capacity
	}
	if idleTimeout <= 0 {
		idleTimeout = defaultIdleTimeout
	}
	p := &Pool{
		clients:     make(chan *clientConn, capacity),
		factory:     factory,
		idleTimeout: idleTimeout,
	}
	for i := 0; i < init; i++ {
		c, err := factory()
		if err != nil {
			return nil, err
		}

		p.clients <- &clientConn{
			conn:     c,
			timeUsed: time.Now(),
		}
	}
	// Fill the rest with empty clients
	for i := 0; i < capacity-init; i++ {
		p.clients <- &clientConn{}
	}
	return p, nil
}

// Get will return the next available client.
// If capacity has not been reached, it will create a new one using the factory.
// Otherwise, it will wait till the next client becomes available or a timeout.
// A timeout of 0 is an indefinite wait.
func (p *Pool) Get(ctx context.Context) (*clientConn, error) {
	clients := p.getClients()
	if clients == nil {
		return nil, fmt.Errorf("pool is closed")
	}

	// Get client
	var client *clientConn
	select {
	case client = <-clients:
		// All good
	case <-ctx.Done():
		return nil, fmt.Errorf("get client from pool timed out")
	}

	// If the client was idle too long, close the connection
	if client.conn != nil && client.timeUsed.Add(p.idleTimeout).Before(time.Now()) {
		client.conn.Close()
		client.conn = nil
	}

	// Create a new connection
	if client.conn == nil {
		var err error
		client.conn, err = p.factory()
		// If there was an error, we put back a placeholder client in the channel
		if err != nil {
			clients <- &clientConn{}
			return nil, err
		}
	}
	return client, nil
}

// Put returns a ClientConn to the pool
func (p *Pool) Put(client *clientConn) error {
	clients := p.getClients()
	if clients == nil {
		return fmt.Errorf("pool is closed")
	}

	client.timeUsed = time.Now()
	select {
	case clients <- client:
		// All good
	default:
		return fmt.Errorf("put a client into a full pool")
	}
	return nil
}

// Close will close all clients. It waits for all clients to be returned.
// The pool channel is then closed, and Get and Put will not be allowed anymore.
func (p *Pool) Close() {
	// Lock to ensure synchronization security in concurrent situations
	p.mu.Lock()
	if p.clients == nil {
		return
	}
	clients := p.clients
	p.clients = nil
	p.mu.Unlock()

	for i := 0; i < cap(clients); i++ {
		client := <-clients
		if client.conn == nil {
			continue
		}
		client.conn.Close()
	}
	close(clients)
}

func (p *Pool) getClients() chan *clientConn {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.clients
}
