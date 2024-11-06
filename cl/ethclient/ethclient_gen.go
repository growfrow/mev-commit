package ethclient

// Code generated by genwrap.go. DO NOT EDIT.

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Client defines all ethereum interfaces used in story.
type Client interface {
	ethereum.BlockNumberReader
	ethereum.ChainIDReader
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ChainSyncReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer
	ethereum.GasPricer1559
	ethereum.LogFilterer
	ethereum.PendingStateReader
	ethereum.TransactionReader
	ethereum.TransactionSender
	HeaderByType(ctx context.Context, typ HeadType) (*types.Header, error)
	EtherBalanceAt(ctx context.Context, addr common.Address) (float64, error)
	PeerCount(ctx context.Context) (uint64, error)
	SetHead(ctx context.Context, height uint64) error
	Address() string
	Close()
}

func (w Wrapper) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	const endpoint = "block_by_hash"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.BlockByHash(ctx, hash)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("BlockByHash, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	const endpoint = "block_by_number"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.BlockByNumber(ctx, number)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("BlockByNumber, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	const endpoint = "header_by_hash"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.HeaderByHash(ctx, hash)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("HeaderByHash, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	const endpoint = "header_by_number"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.HeaderByNumber(ctx, number)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("HeaderByNumber, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	const endpoint = "transaction_count"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.TransactionCount(ctx, blockHash)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("TransactionCount, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	const endpoint = "transaction_in_block"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.TransactionInBlock(ctx, blockHash, index)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("TransactionInBlock, err: %w", err)
	}

	return res0, err
}

// This method subscribes to notifications about changes of the head block of
// the canonical chain.

func (w Wrapper) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	const endpoint = "subscribe_new_head"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.SubscribeNewHead(ctx, ch)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SubscribeNewHead, err: %w", err)
	}

	return res0, err
}

// TransactionByHash checks the pool of pending transactions in addition to the
// blockchain. The isPending return value indicates whether the transaction has been
// mined yet. Note that the transaction may not be part of the canonical chain even if
// it's not pending.

func (w Wrapper) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	const endpoint = "transaction_by_hash"
	defer latency(w.chain, endpoint)()

	res0, res1, err := w.cl.TransactionByHash(ctx, txHash)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("TransactionByHash, err: %w", err)
	}

	return res0, res1, err
}

// TransactionReceipt returns the receipt of a mined transaction. Note that the
// transaction may not be included in the current canonical chain even if a receipt
// exists.

func (w Wrapper) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	const endpoint = "transaction_receipt"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.TransactionReceipt(ctx, txHash)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("TransactionReceipt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	const endpoint = "balance_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("BalanceAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	const endpoint = "storage_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.StorageAt(ctx, account, key, blockNumber)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("StorageAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	const endpoint = "code_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.CodeAt(ctx, account, blockNumber)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("CodeAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	const endpoint = "nonce_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.NonceAt(ctx, account, blockNumber)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("NonceAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	const endpoint = "sync_progress"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.SyncProgress(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SyncProgress, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	const endpoint = "call_contract"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.CallContract(ctx, call, blockNumber)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("CallContract, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	const endpoint = "filter_logs"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.FilterLogs(ctx, q)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("FilterLogs, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	const endpoint = "subscribe_filter_logs"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.SubscribeFilterLogs(ctx, q, ch)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SubscribeFilterLogs, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	const endpoint = "send_transaction"
	defer latency(w.chain, endpoint)()

	err := w.cl.SendTransaction(ctx, tx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SendTransaction, err: %w", err)
	}

	return err
}

func (w Wrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	const endpoint = "suggest_gas_price"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.SuggestGasPrice(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SuggestGasPrice, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	const endpoint = "suggest_gas_tip_cap"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.SuggestGasTipCap(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("SuggestGasTipCap, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	const endpoint = "pending_balance_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.PendingBalanceAt(ctx, account)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("PendingBalanceAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	const endpoint = "pending_storage_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.PendingStorageAt(ctx, account, key)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("PendingStorageAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	const endpoint = "pending_code_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.PendingCodeAt(ctx, account)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("PendingCodeAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	const endpoint = "pending_nonce_at"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.PendingNonceAt(ctx, account)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("PendingNonceAt, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) PendingTransactionCount(ctx context.Context) (uint, error) {
	const endpoint = "pending_transaction_count"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.PendingTransactionCount(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("PendingTransactionCount, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	const endpoint = "estimate_gas"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.EstimateGas(ctx, call)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("EstimateGas, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) BlockNumber(ctx context.Context) (uint64, error) {
	const endpoint = "block_number"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.BlockNumber(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("BlockNumber, err: %w", err)
	}

	return res0, err
}

func (w Wrapper) ChainID(ctx context.Context) (*big.Int, error) {
	const endpoint = "chain_id"
	defer latency(w.chain, endpoint)()

	res0, err := w.cl.ChainID(ctx)
	if err != nil {
		incError(w.chain, endpoint)
		err = fmt.Errorf("ChainID, err: %w", err)
	}

	return res0, err
}
