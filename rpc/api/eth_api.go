package api

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/tendermint/tendermint/libs/log"
	tmrpc "github.com/tendermint/tendermint/rpc/core"

	"github.com/smartbch/moeingevm/ebp"
	"github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	"github.com/smartbch/smartbch/internal/ethutils"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

const (
	// DefaultGasPrice is default gas price for evm transactions
	DefaultGasPrice = 20000000000
	// DefaultRPCGasLimit is default gas limit for RPC call operations
	DefaultRPCGasLimit = 10000000
)

var _ PublicEthAPI = (*ethAPI)(nil)

type PublicEthAPI interface {
	Accounts() ([]common.Address, error)
	BlockNumber() (hexutil.Uint64, error)
	Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumber) (hexutil.Bytes, error)
	ChainId() hexutil.Uint64
	Coinbase() (common.Address, error)
	EstimateGas(args rpctypes.CallArgs) (hexutil.Uint64, error)
	GasPrice() *hexutil.Big
	GetBalance(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Big, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum gethrpc.BlockNumber) *hexutil.Uint
	GetCode(addr common.Address, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error)
	GetStorageAt(addr common.Address, key string, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.Transaction, error)
	GetTransactionByBlockNumberAndIndex(blockNum gethrpc.BlockNumber, idx hexutil.Uint) (*rpctypes.Transaction, error)
	GetTransactionByHash(hash common.Hash) (*rpctypes.Transaction, error)
	GetTransactionCount(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Uint64, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{}
	GetUncleByBlockNumberAndIndex(number hexutil.Uint, idx hexutil.Uint) map[string]interface{}
	GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint
	GetUncleCountByBlockNumber(_ gethrpc.BlockNumber) hexutil.Uint
	ProtocolVersion() hexutil.Uint
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error) // ?
	SendTransaction(args rpctypes.SendTxArgs) (common.Hash, error)
	Syncing() (interface{}, error)
}

type ethAPI struct {
	backend  sbchapi.BackendService
	accounts map[common.Address]*ecdsa.PrivateKey // only for test
	logger   log.Logger
}

func newEthAPI(backend sbchapi.BackendService, testKeys []string, logger log.Logger) *ethAPI {
	return &ethAPI{
		backend:  backend,
		accounts: loadTestAccounts(testKeys, logger),
		logger:   logger.With("module", "eth-api"),
	}
}

func loadTestAccounts(testKeys []string, logger log.Logger) map[common.Address]*ecdsa.PrivateKey {
	accs := make(map[common.Address]*ecdsa.PrivateKey, len(testKeys))
	for _, testKey := range testKeys {
		if key, _, err := ethutils.HexToPrivKey(testKey); err == nil {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			accs[addr] = key
		} else {
			logger.Error("failed to load private key:", testKey, err.Error())
		}
	}
	return accs
}

func (api *ethAPI) Accounts() ([]common.Address, error) {
	addrs := make([]common.Address, 0, len(api.accounts))
	for addr := range api.accounts {
		addrs = append(addrs, addr)
	}

	sort.Slice(addrs, func(i, j int) bool {
		for k := 0; k < common.AddressLength; k++ {
			if addrs[i][k] < addrs[j][k] {
				return true
			} else if addrs[i][k] > addrs[j][k] {
				return false
			}
		}
		return false
	})
	return addrs, nil
}

// https://eth.wiki/json-rpc/API#eth_blockNumber
func (api *ethAPI) BlockNumber() (hexutil.Uint64, error) {
	return hexutil.Uint64(api.backend.LatestHeight()), nil
}

// https://eips.ethereum.org/EIPS/eip-695
func (api *ethAPI) ChainId() hexutil.Uint64 {
	chainID := api.backend.ChainId()
	return hexutil.Uint64(chainID.Uint64())
}

// https://eth.wiki/json-rpc/API#eth_coinbase
func (api *ethAPI) Coinbase() (common.Address, error) {
	// TODO: this is temporary implementation
	return common.Address{}, nil
}

// https://eth.wiki/json-rpc/API#eth_gasPrice
func (api *ethAPI) GasPrice() *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(0))
}

// https://eth.wiki/json-rpc/API#eth_getBalance
func (api *ethAPI) GetBalance(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Big, error) {
	// ignore blockNumber temporary
	b, err := api.backend.GetBalance(addr, int64(gethrpc.LatestBlockNumber))
	if err != nil {
		if err == types.ErrAccNotFound {
			return (*hexutil.Big)(big.NewInt(0)), nil
		}
		return nil, err
	}
	return (*hexutil.Big)(b), err
}

// https://eth.wiki/json-rpc/API#eth_getCode
func (api *ethAPI) GetCode(addr common.Address, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error) {
	// ignore blockNumber temporary
	code, _ := api.backend.GetCode(addr, int64(gethrpc.LatestBlockNumber))
	return code, nil
}

// https://eth.wiki/json-rpc/API#eth_getStorageAt
func (api *ethAPI) GetStorageAt(addr common.Address, key string, blockNum gethrpc.BlockNumber) (hexutil.Bytes, error) {
	// ignore blockNumber temporary
	hash := common.HexToHash(key)
	key = string(hash[:])
	return api.backend.GetStorageAt(addr, key, int64(gethrpc.LatestBlockNumber)), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockByHash
func (api *ethAPI) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}

	var txs []*types.Transaction
	if fullTx {
		txs, err = api.backend.GetTxListByHeight(uint32(block.Number))
		if err != nil {
			return nil, err
		}
	}

	return blockToRpcResp(block, txs), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockByNumber
func (api *ethAPI) GetBlockByNumber(blockNum gethrpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		if err == types.ErrBlockNotFound {
			return nil, nil
		}
		return nil, err
	}

	var txs []*types.Transaction
	if fullTx {
		txs, err = api.backend.GetTxListByHeight(uint32(block.Number))
		if err != nil {
			return nil, err
		}
	}
	return blockToRpcResp(block, txs), nil
}

// https://eth.wiki/json-rpc/API#eth_getBlockTransactionCountByHash
func (api *ethAPI) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions))
	return &n
}

// https://eth.wiki/json-rpc/API#eth_getBlockTransactionCountByNumber
func (api *ethAPI) GetBlockTransactionCountByNumber(blockNum gethrpc.BlockNumber) *hexutil.Uint {
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions))
	return &n
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByBlockHashAndIndex
func (api *ethAPI) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	block, err := api.backend.BlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return api.getTxByIdx(block, idx)
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByBlockNumberAndIndex
func (api *ethAPI) GetTransactionByBlockNumberAndIndex(blockNum gethrpc.BlockNumber, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	block, err := api.getBlockByNum(blockNum)
	if err != nil {
		return nil, err
	}
	return api.getTxByIdx(block, idx)
}

// https://eth.wiki/json-rpc/API#eth_getTransactionByHash
func (api *ethAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.Transaction, error) {
	tx, _, _, _, err := api.backend.GetTransaction(hash)
	if err != nil {
		return nil, nil
	}
	return txToRpcResp(tx), nil
}

// https://eth.wiki/json-rpc/API#eth_getTransactionCount
func (api *ethAPI) GetTransactionCount(addr common.Address, blockNum gethrpc.BlockNumber) (*hexutil.Uint64, error) {
	// ignore blockNumber temporary
	nonce, err := api.backend.GetNonce(addr)
	if err != nil {
		return nil, err
	}
	nonceU64 := hexutil.Uint64(nonce)
	return &nonceU64, nil
}

func (api *ethAPI) getBlockByNum(blockNum gethrpc.BlockNumber) (*types.Block, error) {
	height := blockNum.Int64()
	if height <= 0 {
		// get latest block height
		return api.backend.CurrentBlock()
	}
	return api.backend.BlockByNumber(height)
}

func (api *ethAPI) getTxByIdx(block *types.Block, idx hexutil.Uint) (*rpctypes.Transaction, error) {
	if uint64(idx) >= uint64(len(block.Transactions)) {
		// return if index out of bounds
		return nil, nil
	}

	txHash := block.Transactions[idx]
	tx, _, _, _, err := api.backend.GetTransaction(txHash)
	if err != nil {
		return nil, err
	}

	return txToRpcResp(tx), nil
}

// https://eth.wiki/json-rpc/API#eth_getTransactionReceipt
func (api *ethAPI) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	tx, _, _, _, err := api.backend.GetTransaction(hash)
	if err != nil {
		// the transaction is not yet available
		return nil, nil
	}
	return txToReceiptRpcResp(tx), nil
}

// https://eth.wiki/json-rpc/API#eth_getUncleByBlockHashAndIndex
func (api *ethAPI) GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{} {
	// not supported
	return nil
}

// https://eth.wiki/json-rpc/API#eth_getUncleByBlockNumberAndIndex
func (api *ethAPI) GetUncleByBlockNumberAndIndex(number hexutil.Uint, idx hexutil.Uint) map[string]interface{} {
	// not supported
	return nil
}

// https://eth.wiki/json-rpc/API#
func (api *ethAPI) GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint {
	// not supported
	return 0
}

// https://eth.wiki/json-rpc/API#eth_getUncleCountByBlockHash
func (api *ethAPI) GetUncleCountByBlockNumber(_ gethrpc.BlockNumber) hexutil.Uint {
	// not supported
	return 0
}

// https://eth.wiki/json-rpc/API#eth_protocolVersion
func (api *ethAPI) ProtocolVersion() hexutil.Uint {
	return hexutil.Uint(api.backend.ProtocolVersion())
}

// https://eth.wiki/json-rpc/API#eth_sendRawTransaction
func (api *ethAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	tx, err := ethutils.DecodeTx(data)
	if err != nil {
		return common.Hash{}, err
	}

	tmTxHash, err := api.backend.SendRawTx(data)
	if err != nil {
		return tmTxHash, err
	}

	return tx.Hash(), nil
}

// https://eth.wiki/json-rpc/API#eth_sendTransaction
func (api *ethAPI) SendTransaction(args rpctypes.SendTxArgs) (common.Hash, error) {
	privKey, found := api.accounts[args.From]
	if !found {
		return common.Hash{}, errors.New("unknown account: " + args.From.Hex())
	}

	if args.Nonce == nil {
		if nonce, err := api.backend.GetNonce(args.From); err == nil {
			args.Nonce = (*hexutil.Uint64)(&nonce)
		}
	}

	tx, err := createGethTxFromSendTxArgs(args)
	if err != nil {
		return common.Hash{}, err
	}

	chainID := api.backend.ChainId()
	tx, err = ethutils.SignTx(tx, chainID, privKey)
	if err != nil {
		return common.Hash{}, err
	}

	txBytes, err := ethutils.EncodeTx(tx)
	if err != nil {
		return common.Hash{}, err
	}

	tmTxHash, err := api.backend.SendRawTx(txBytes)
	if err != nil {
		return tmTxHash, err
	}

	txHash := tx.Hash()
	return txHash, err
}

// https://eth.wiki/json-rpc/API#eth_syncing
func (api *ethAPI) Syncing() (interface{}, error) {
	status, err := tmrpc.Status(nil)
	if err != nil {
		return false, err
	}
	if !status.SyncInfo.CatchingUp {
		return false, nil
	}

	return map[string]interface{}{
		// "startingBlock": nil, // NA
		"currentBlock": hexutil.Uint64(status.SyncInfo.LatestBlockHeight),
		// "highestBlock":  nil, // NA
		// "pulledStates":  nil, // NA
		// "knownStates":   nil, // NA
	}, nil
}

// https://eth.wiki/json-rpc/API#eth_call
func (api *ethAPI) Call(args rpctypes.CallArgs, blockNr gethrpc.BlockNumber) (hexutil.Bytes, error) {
	// ignore blockNumber temporary
	tx, from, err := api.createGethTxFromCallArgs(args)
	if err != nil {
		return hexutil.Bytes{}, err
	}

	statusCode, retData := api.backend.Call(tx, from)
	if !ebp.StatusIsFailure(statusCode) {
		return retData, nil
	}

	return nil, toCallErr(statusCode, retData)
}

// https://eth.wiki/json-rpc/API#eth_estimateGas
func (api *ethAPI) EstimateGas(args rpctypes.CallArgs) (hexutil.Uint64, error) {
	tx, from, err := api.createGethTxFromCallArgs(args)
	if err != nil {
		return 0, err
	}

	statusCode, retData, gas := api.backend.EstimateGas(tx, from)
	if !ebp.StatusIsFailure(statusCode) {
		return hexutil.Uint64(gas), nil
	}

	return 0, toCallErr(statusCode, retData)
}

func (api *ethAPI) createGethTxFromCallArgs(args rpctypes.CallArgs,
) (*gethtypes.Transaction, common.Address, error) {

	var from, to common.Address
	if args.From != nil {
		from = *args.From
	}
	if args.To != nil {
		to = *args.To
	}

	var val *big.Int
	if args.Value != nil {
		val = args.Value.ToInt()
	} else {
		val = big.NewInt(0)
	}

	var gasLimit uint64 = DefaultRPCGasLimit
	if args.Gas != nil {
		gasLimit = uint64(*args.Gas)
	}

	var gasPrice *big.Int
	if args.GasPrice != nil {
		gasPrice = args.GasPrice.ToInt()
	} else {
		gasPrice = big.NewInt(DefaultGasPrice)
	}

	var data []byte
	if args.Data != nil {
		data = *args.Data
	}

	//nonce, err := api.GetTransactionCount(from, blockNr)
	//if err != nil {
	//	return nil, from, err
	//}

	// TODO: replace with ethutils.NewTx()
	tx := gethtypes.NewTransaction(0, to, val, gasLimit, gasPrice, data)
	return tx, from, nil
}
