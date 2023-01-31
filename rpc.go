package go_eth_client

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
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
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/go-eth-client/utils"
)

var _ Client = (*EthRPC)(nil)

const (
	defaultPoolSize        = 6               // 连接池默认大小
	defaultPoolInit        = 4               // 连接池默认初始连接数
	defaultPoolIdleTimeout = 1 * time.Hour   // 连接池中连接的默认闲置时间阈值
	defaultCallTimeout     = 6 * time.Second // 默认请求超时时间

	waitReceipt = 300 * time.Millisecond
)

type EthRPC struct {
	urls            []string          // bitxhub各节点的URL
	privateKey      *ecdsa.PrivateKey // 用于交易签名的默认私钥
	cid             *big.Int          // ChainID
	pool            *Pool             // 客户端连接池
	poolSize        int               // 连接池大小
	poolInit        int               // 连接池初始连接数
	poolIdleTimeout time.Duration     // 连接池中连接的闲置时间阈值
	callTimeout     time.Duration     // 请求的超时时间（包括等待连接和json-rpc请求的超时时间总和）
	logger          Logger
}

type Option func(*EthRPC)

func WithUrls(urls []string) Option {
	return func(config *EthRPC) {
		config.urls = urls
	}
}

func WithPriKey(pk *ecdsa.PrivateKey) Option {
	return func(config *EthRPC) {
		config.privateKey = pk
	}
}

func WithPoolSize(poolSize int) Option {
	return func(config *EthRPC) {
		config.poolSize = poolSize
	}
}

func WithPoolInit(poolInit int) Option {
	return func(config *EthRPC) {
		config.poolInit = poolInit
	}
}

func WithPoolIdleTimeout(t time.Duration) Option {
	return func(config *EthRPC) {
		config.poolIdleTimeout = t
	}
}

func WithCallTimeout(t time.Duration) Option {
	return func(config *EthRPC) {
		config.callTimeout = t
	}
}

func WithLogger(logger Logger) Option {
	return func(config *EthRPC) {
		config.logger = logger
	}
}

func New(opts ...Option) (*EthRPC, error) {
	// initialize config
	rpc := &EthRPC{}
	for _, opt := range opts {
		opt(rpc)
	}

	// check and set config
	if len(rpc.urls) == 0 {
		return nil, fmt.Errorf("bitxhub urls cant not be 0")
	}
	if rpc.poolSize <= 0 {
		rpc.poolSize = defaultPoolSize
	}
	if rpc.poolInit <= 0 {
		rpc.poolInit = defaultPoolInit
	}
	if rpc.poolIdleTimeout <= 0 {
		rpc.poolIdleTimeout = defaultPoolIdleTimeout
	}
	if rpc.callTimeout <= 0 {
		rpc.callTimeout = defaultCallTimeout
	}
	if rpc.logger == nil {
		rpc.logger = log.NewWithModule("go-eth-client")
	}

	// generate other config
	var err error
	rpc.pool, err = NewPool(rpc.newClient, rpc.poolInit, rpc.poolSize, rpc.poolIdleTimeout)
	if err != nil {
		return nil, err
	}
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		rpc.cid, err = client.conn.ChainID(ctx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return rpc, nil
}

func (rpc *EthRPC) newClient() (*ethclient.Client, string, error) {
	randIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(rpc.urls))
	// Dial can't create connection, only create an instance
	client, err := ethclient.Dial(rpc.urls[randIndex])
	if err != nil {
		rpc.logger.Errorf("Dial url %s failed", rpc.urls[randIndex])
		return nil, "", fmt.Errorf("dial url %s failed", rpc.urls[randIndex])
	}
	rpc.logger.Debugf("Create instance that dial with %s successfully", rpc.urls[randIndex])
	return client, rpc.urls[randIndex], nil
}

func (rpc *EthRPC) putClient(client *clientConn) {
	if err := rpc.pool.Put(client); err != nil {
		rpc.logger.Errorf("Put into pool err: %s", err)
	}
}

func (rpc *EthRPC) wrapper(f func(ctx context.Context, client *clientConn) error) error {
	var otherErr error
	if err := retry.Retry(func(attempt uint) error {
		ctx, cancel := context.WithTimeout(context.Background(), rpc.callTimeout)
		defer cancel()
		client, err := rpc.pool.Get(ctx)
		if err != nil {
			return err
		}
		defer rpc.putClient(client)
		if err := retry.Retry(func(attempt uint) error {
			ctx, cancel := context.WithTimeout(context.Background(), rpc.callTimeout)
			defer cancel()
			if err := f(ctx, client); err != nil {
				rpc.logger.Warning(err.Error())
				// if error is 'connection refused', retry
				if strings.Contains(err.Error(), "connection refused") {
					return err
				}
				otherErr = err
			}
			return nil
		}, strategy.Wait(200*time.Millisecond), strategy.Limit(3)); err != nil {
			// if still failed after retry 5 times, close the client
			client.Close()
			rpc.logger.Errorf("close connection with %s", client.url)
			return err
		}
		return nil
	}, strategy.Wait(1*time.Second), strategy.Limit(uint(2*len(rpc.urls)))); err != nil {
		return err
	}

	if otherErr != nil {
		return otherErr
	}
	return nil
}

func (rpc *EthRPC) EthEstimateGas(msg ethereum.CallMsg) (uint64, error) {
	var estimateGas uint64
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		estimateGas, err = client.conn.EstimateGas(ctx, msg)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return estimateGas, nil
}

func (rpc *EthRPC) EthGetTransactionByHash(txHash common.Hash) (*types.Transaction, error) {
	var tx *types.Transaction
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		tx, _, err = client.conn.TransactionByHash(ctx, txHash)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return tx, nil
}

func (rpc *EthRPC) EthGetTransactionByBlockHashAndIndex(blockHash common.Hash, index int) (*types.Transaction, error) {
	var tx *types.Transaction
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		tx, err = client.conn.TransactionInBlock(ctx, blockHash, uint(index))
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return tx, nil
}

func (rpc *EthRPC) EthGetTransactionByBlockNumberAndIndex(blockNumber *big.Int, index int) (*types.Transaction, error) {
	var block *types.Block
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		block, err = client.conn.BlockByNumber(ctx, blockNumber)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return block.Transactions()[index], nil
}

func (rpc *EthRPC) EthGetBlockTransactionCountByHash(blockHash common.Hash) (uint64, error) {
	var num uint
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		num, err = client.conn.TransactionCount(ctx, blockHash)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return uint64(num), nil
}

func (rpc *EthRPC) EthGetBlockTransactionCountByNumber(blockNumber *big.Int) (uint64, error) {
	var block *types.Block
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		block, err = client.conn.BlockByNumber(ctx, blockNumber)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
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
	// set transaction options
	transactionOpts := &TransactionOptions{}
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

	var (
		address common.Address
		tx      *types.Transaction
		receipt *types.Receipt
	)
	// deploy contract
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		address, tx, _, err = bind.DeployContract(txOpts, abi, common.FromHex(code), client.conn, args...)
		if err != nil {
			return err
		}
		time.Sleep(waitReceipt)
		if err := retry.Retry(func(attempt uint) error {
			receipt, err = client.conn.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return err
			}
			return nil
		}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond))); err != nil {
			return err
		}
		return nil
	}); err != nil {
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

		var (
			address common.Address
			tx      *types.Transaction
			receipt *types.Receipt
		)
		// deploy contract
		if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
			var err error
			address, tx, _, err = bind.DeployContract(txOpts, parsed, common.FromHex(code), client.conn, args...)
			if err != nil {
				return err
			}
			time.Sleep(waitReceipt)
			if err := retry.Retry(func(attempt uint) error {
				receipt, err = client.conn.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					return err
				}
				return nil
			}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond))); err != nil {
				return err
			}
			return nil
		}); err != nil {
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
	var output []byte
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		output, err = client.conn.CallContract(ctx, msg, nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
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
		var output []byte
		if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
			var err error
			output, err = client.conn.CallContract(ctx, msg, nil)
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
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
	var price *big.Int
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		price, err = client.conn.SuggestGasPrice(ctx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return price, nil
}

func (rpc *EthRPC) EthGetTransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	var (
		receipt *types.Receipt
		err     error
	)
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		if err := retry.Retry(func(attempt uint) error {
			receipt, err = client.conn.TransactionReceipt(ctx, hash)
			if err != nil {
				return err
			}
			return nil
		}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond))); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return receipt, nil
}

func (rpc *EthRPC) EthGetTransactionCount(account common.Address, blockNumber *big.Int) (uint64, error) {
	var nonce uint64
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		nonce, err = client.conn.NonceAt(ctx, account, blockNumber)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return nonce, nil
}

func (rpc *EthRPC) EthGetBlockByNumber(blockNumber *big.Int, fullTx bool) (*types.Block, error) {
	var (
		err   error
		block *types.Block
	)
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		if !fullTx {
			blockHeader, err := client.conn.HeaderByNumber(ctx, blockNumber)
			if err != nil {
				return err
			}
			block = types.NewBlockWithHeader(blockHeader)
			return nil
		}
		block, err = client.conn.BlockByNumber(ctx, blockNumber)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return block, nil
}

func (rpc *EthRPC) EthGetBalance(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var balance *big.Int
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		balance, err = client.conn.BalanceAt(ctx, account, blockNumber)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return balance, nil
}

func (rpc *EthRPC) EthSendTransaction(privKey *ecdsa.PrivateKey, transaction *types.Transaction) (common.Hash, error) {
	signTx, err := types.SignTx(transaction, types.NewEIP155Signer(rpc.cid), privKey)
	if err != nil {
		return common.Hash{}, err
	}
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		err := client.conn.SendTransaction(ctx, signTx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), nil
}

func (rpc *EthRPC) EthSendRawTransaction(transaction *types.Transaction) (common.Hash, error) {
	if err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		err := client.conn.SendTransaction(ctx, transaction)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
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
	var code []byte
	err := rpc.wrapper(func(ctx context.Context, client *clientConn) error {
		var err error
		code, err = client.conn.CodeAt(ctx, account, blockNumber)
		if err != nil {
			return err
		}
		return nil
	})
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
