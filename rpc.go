package go_eth_client

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/meshplus/go-eth-client/utils"
)

var _ Client = (*EthRPC)(nil)

const (
	poolSize        = 6             // 连接池默认大小
	poolInit        = 4             // 连接池默认初始连接数
	poolIdleTimeout = 1 * time.Hour // 连接池中连接的默认闲置时间阈值

	waitReceipt = 300 * time.Millisecond
)

type EthRPC struct {
	urls       []string          // bitxhub各节点的URL
	pool       *Pool             // 客户端连接池
	poolSize   int               // 连接池大小
	privateKey *ecdsa.PrivateKey // 用于交易签名的默认私钥
	cid        *big.Int          // ChainID
}

type Option func(*EthRPC)

func WithUrls(urls []string) Option {
	return func(config *EthRPC) {
		config.urls = urls
	}
}

func WithPoolSize(poolSize int) Option {
	return func(config *EthRPC) {
		config.poolSize = poolSize
	}
}

func WithPriKey(pk *ecdsa.PrivateKey) Option {
	return func(config *EthRPC) {
		config.privateKey = pk
	}
}

func New(opts ...Option) (*EthRPC, error) {
	// initialize config
	rpc := &EthRPC{}
	for _, opt := range opts {
		opt(rpc)
	}

	// check
	if len(rpc.urls) == 0 {
		return nil, fmt.Errorf("bitxhub urls cant not be 0")
	}
	if rpc.poolSize <= 0 {
		rpc.poolSize = poolSize
	}

	// generate other config
	var err error
	rpc.pool, err = NewPool(rpc.newClient, poolInit, rpc.poolSize, poolIdleTimeout)
	if err != nil {
		return nil, err
	}
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	rpc.cid, err = client.conn.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	return rpc, nil
}

func (rpc *EthRPC) newClient() (*ethclient.Client, error) {
	var client *ethclient.Client
	if err := retry.Retry(func(attempt uint) error {
		var err error
		randIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(rpc.urls))
		client, err = ethclient.Dial(rpc.urls[randIndex])
		if err != nil {
			return fmt.Errorf("dial url %s failed", rpc.urls[randIndex])
		}
		return nil
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(uint(5*len(rpc.urls)))); err != nil {
		return nil, err
	}
	return client, nil
}

func (rpc *EthRPC) putClient(client *clientConn) {
	if err := rpc.pool.Put(client); err != nil {
		log.Printf("put into pool err: %s", err)
	}
}

func (rpc *EthRPC) EthEstimateGas(msg ethereum.CallMsg) (uint64, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return 0, err
	}
	defer rpc.putClient(client)
	estimateGas, err := client.conn.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}
	return estimateGas, nil
}

func (rpc *EthRPC) EthGetTransactionByHash(txHash common.Hash) (*types.Transaction, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	tx, _, err := client.conn.TransactionByHash(context.Background(), txHash)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (rpc *EthRPC) EthGetTransactionByBlockHashAndIndex(blockHash common.Hash, index int) (*types.Transaction, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	tx, err := client.conn.TransactionInBlock(context.Background(), blockHash, uint(index))
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (rpc *EthRPC) EthGetTransactionByBlockNumberAndIndex(blockNumber *big.Int, index int) (*types.Transaction, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	block, err := client.conn.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	return txs[index], nil
}

func (rpc *EthRPC) EthGetBlockTransactionCountByHash(blockHash common.Hash) (uint64, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return 0, err
	}
	defer rpc.putClient(client)
	num, err := client.conn.TransactionCount(context.Background(), blockHash)
	if err != nil {
		return 0, err
	}
	return uint64(num), nil
}

func (rpc *EthRPC) EthGetBlockTransactionCountByNumber(blockNumber *big.Int) (uint64, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return 0, err
	}
	defer rpc.putClient(client)
	block, err := client.conn.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return 0, err
	}
	return uint64(block.Transactions().Len()), nil
}

func (rpc *EthRPC) Compile(sourceFiles ...string) (*CompileResult, error) {
	contracts, err := compiler.CompileSolidity("", sourceFiles...)
	if err != nil {
		return nil, fmt.Errorf("compile contract: %w", err)
	}
	var abis, bins, names []string
	for name, contract := range contracts {
		contractAbi, err := json.Marshal(contract.Info.AbiDefinition)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABIs from compiler output: %w", err)
		}
		abis = append(abis, string(contractAbi))
		bins = append(bins, contract.Code)
		names = append(names, name)
	}
	return &CompileResult{
		Abi:   abis,
		Bin:   bins,
		Names: names,
	}, nil
}

func (rpc *EthRPC) DeployByCode(privKey *ecdsa.PrivateKey, abi abi.ABI, code string, args []interface{}, opts ...TransactionOption) (string, uint64, error) {
	transactionOpts := &TransactionOptions{}
	// set transaction options
	for _, opt := range opts {
		opt(transactionOpts)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, rpc.cid)
	if err != nil {
		return "", 0, err
	}
	txOpts.GasPrice = transactionOpts.GasPrice

	if transactionOpts.Nonce == 0 {
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privKey.PublicKey), nil)
		if err != nil {
			return "", 0, err
		}
		transactionOpts.Nonce = nonce
	}
	txOpts.Nonce = big.NewInt(int64(transactionOpts.Nonce))
	if transactionOpts.GasLimit == 0 {
		txOpts.GasLimit = 100000000
	} else {
		txOpts.GasLimit = transactionOpts.GasLimit
	}

	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return "", 0, err
	}
	defer rpc.putClient(client)

	address, tx, _, err := bind.DeployContract(txOpts, abi, common.FromHex(code), client.conn, args...)
	if err != nil {
		return "", 0, err
	}
	// try three times
	var receipt *types.Receipt
	if err := retry.Retry(func(attempt uint) error {
		receipt, err = client.conn.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			return err
		}
		return nil
	}, strategy.Wait(2*time.Second), strategy.Limit(5)); err != nil {
		return "", 0, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return "", 0, fmt.Errorf("deploy contract failed, tx hash is: %s", tx.Hash())
	}
	return address.String(), receipt.BlockNumber.Uint64(), nil
}

func (rpc *EthRPC) Deploy(privKey *ecdsa.PrivateKey, result *CompileResult, args []interface{}, opts ...TransactionOption) ([]string, error) {
	if len(result.Abi) == 0 || len(result.Bin) == 0 || len(result.Names) == 0 {
		return nil, fmt.Errorf("empty contract")
	}

	transactionOpts := &TransactionOptions{}
	// set transaction options
	for _, opt := range opts {
		opt(transactionOpts)
	}
	// load privateKey
	if transactionOpts.PrivateKey != nil {
		privKey = transactionOpts.PrivateKey
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, rpc.cid)
	if err != nil {
		return nil, err
	}
	txOpts.GasPrice = transactionOpts.GasPrice

	if transactionOpts.Nonce == 0 {
		nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privKey.PublicKey), nil)
		if err != nil {
			return nil, err
		}
		transactionOpts.Nonce = nonce
	}

	txOpts.Nonce = big.NewInt(int64(transactionOpts.Nonce))
	if transactionOpts.GasLimit == 0 {
		txOpts.GasLimit = 100000000
	} else {
		txOpts.GasLimit = transactionOpts.GasLimit
	}

	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)

	addresses := make([]string, 0)
	for i, bin := range result.Bin {
		if bin == "0x" {
			continue
		}
		parsed, err := abi.JSON(strings.NewReader(result.Abi[i]))
		if err != nil {
			return nil, err
		}
		code := strings.TrimPrefix(strings.TrimSpace(bin), "0x")

		address, tx, _, err := bind.DeployContract(txOpts, parsed, common.FromHex(code), client.conn, args...)
		if err != nil {
			return nil, err
		}
		// try three times
		time.Sleep(waitReceipt)
		var receipt *types.Receipt
		if err := retry.Retry(func(attempt uint) error {
			receipt, err = client.conn.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				return err
			}
			return nil
		}, strategy.Wait(1*time.Second), strategy.Limit(3)); err != nil {
			return nil, err
		}
		if receipt.Status == types.ReceiptStatusFailed {
			return nil, fmt.Errorf("deploy contract failed, tx hash is: %s", tx.Hash())
		}
		addresses = append(addresses, address.String())
	}
	return addresses, nil
}

func (rpc *EthRPC) EthCall(contractAbi *abi.ABI, address string, method string, args []interface{}) ([]interface{}, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)

	var invokeRes []interface{}
	to := common.HexToAddress(address)
	packed, err := contractAbi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{To: &to, Data: packed}
	if !contractAbi.Methods[method].IsConstant() {
		return nil, fmt.Errorf("EthCall function need the method is read-only")
	}
	output, err := client.conn.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}
	if len(output) == 0 {
		if code, err := rpc.EthGetCode(to, nil); err != nil {
			return nil, err
		} else if code == "0x" {
			return nil, fmt.Errorf("no code at your contract addresss")
		}
		return nil, fmt.Errorf("output is empty")
	}
	// unpack result for display
	invokeRes, err = utils.UnpackOutput(contractAbi, method, string(output))
	if err != nil {
		return nil, err
	}
	return invokeRes, nil
}

func (rpc *EthRPC) Invoke(privKey *ecdsa.PrivateKey, contractAbi *abi.ABI, address string, method string, args []interface{}, opts ...TransactionOption) ([]interface{}, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)

	var invokeRes []interface{}
	txOpts := &TransactionOptions{}
	for _, opt := range opts {
		opt(txOpts)
	}
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	to := common.HexToAddress(address)
	packed, err := contractAbi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: from, To: &to, Data: packed}
	if contractAbi.Methods[method].IsConstant() {
		output, err := client.conn.CallContract(context.Background(), msg, nil)
		if err != nil {
			return nil, err
		}
		if len(output) == 0 {
			if code, err := rpc.EthGetCode(to, nil); err != nil {
				return nil, err
			} else if code == "0x" {
				return nil, fmt.Errorf("no code at your contract addresss")
			}
			return nil, fmt.Errorf("output is empty")
		}
		// unpack result for display
		invokeRes, err = utils.UnpackOutput(contractAbi, method, string(output))
		if err != nil {
			return nil, err
		}
		return invokeRes, nil
	} else {
		if txOpts.Nonce == 0 {
			nonce, err := rpc.EthGetTransactionCount(crypto.PubkeyToAddress(privKey.PublicKey), nil)
			if err != nil {
				return nil, err
			}
			txOpts.Nonce = nonce
		}
		if txOpts.GasLimit == 0 {
			txOpts.GasLimit = 1000000
		}
		if txOpts.GasPrice == nil {
			price, err := rpc.EthGasPrice()
			if err != nil {
				return nil, err
			}
			txOpts.GasPrice = price
		}
		tx := utils.NewTransaction(txOpts.Nonce, to, txOpts.GasLimit, txOpts.GasPrice, packed, nil)
		receipt, err := rpc.EthSendTransactionWithReceipt(privKey, tx)
		if err != nil {
			return nil, fmt.Errorf("invoke err:%s", err)
		}
		return []interface{}{receipt}, nil
	}
}

func (rpc *EthRPC) EthGasPrice() (*big.Int, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	price, err := client.conn.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	return price, nil
}

func (rpc *EthRPC) EthGetTransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	var (
		receipt    *types.Receipt
		err        error
		otherError error
	)

	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)

	err = retry.Retry(func(attempt uint) error {
		receipt, err = client.conn.TransactionReceipt(context.Background(), hash)
		if err != nil {
			return err
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))
	if err != nil {
		return nil, err
	}
	if otherError != nil {
		return nil, otherError
	}
	return receipt, nil
}

func (rpc *EthRPC) EthGetTransactionCount(account common.Address, blockNumber *big.Int) (uint64, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return 0, err
	}
	defer rpc.putClient(client)
	nonce, err := client.conn.NonceAt(context.Background(), account, blockNumber)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (rpc *EthRPC) EthGetBlockByNumber(blockNumber *big.Int, fullTx bool) (*types.Block, error) {
	var (
		err   error
		block *types.Block
	)
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	if !fullTx {
		blockHeader, err := client.conn.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			return nil, err
		}
		return types.NewBlockWithHeader(blockHeader), nil
	}
	block, err = client.conn.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (rpc *EthRPC) EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	defer rpc.putClient(client)
	balance, err := client.conn.BalanceAt(context.Background(), account, blockNumber)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (rpc *EthRPC) EthSendTransaction(privKey *ecdsa.PrivateKey, transaction *types.Transaction) (common.Hash, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return common.Hash{}, err
	}
	defer rpc.putClient(client)
	signTx, err := types.SignTx(transaction, types.NewEIP155Signer(rpc.cid), privKey)
	if err != nil {
		return common.Hash{}, err
	}
	err = client.conn.SendTransaction(context.Background(), signTx)
	if err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), nil
}

func (rpc *EthRPC) EthSendRawTransaction(transaction *types.Transaction) (common.Hash, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return common.Hash{}, err
	}
	defer rpc.putClient(client)
	err = client.conn.SendTransaction(context.Background(), transaction)
	if err != nil {
		return common.Hash{}, err
	}
	return transaction.Hash(), nil
}

func (rpc *EthRPC) EthSendTransactionWithReceipt(privKey *ecdsa.PrivateKey, transaction *types.Transaction) (*types.Receipt, error) {
	hash, err := rpc.EthSendTransaction(privKey, transaction)
	if err != nil {
		return nil, err
	}
	time.Sleep(waitReceipt)
	receipt, err := rpc.EthGetTransactionReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (rpc *EthRPC) EthSendRawTransactionWithReceipt(transaction *types.Transaction) (*types.Receipt, error) {
	hash, err := rpc.EthSendRawTransaction(transaction)
	if err != nil {
		return nil, err
	}

	time.Sleep(waitReceipt)
	receipt, err := rpc.EthGetTransactionReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (rpc *EthRPC) EthGetCode(account common.Address, blockNumber *big.Int) (string, error) {
	client, err := rpc.pool.Get(context.Background())
	if err != nil {
		return "", err
	}
	defer rpc.putClient(client)
	code, err := client.conn.CodeAt(context.Background(), account, blockNumber)
	if err != nil || len(code) == 0 {
		return "0x", err
	}
	return common.Bytes2Hex(code), nil
}

func (rpc *EthRPC) EthGetChainId() *big.Int {
	return rpc.cid
}

func (rpc *EthRPC) Stop() {
	if rpc.pool == nil {
		return
	}
	rpc.pool.Close()
	rpc.pool = nil
}
