package dex

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"liquidityScope/internal/model"
)

func TestV3PoolDecoderSwap(t *testing.T) {
	poolABI, err := V3PoolABI()
	if err != nil {
		t.Fatalf("abi parse: %v", err)
	}

	pool := common.HexToAddress("0x1111111111111111111111111111111111111111")
	poolMetaCache := NewPoolMetaCache()
	poolMetaCache.Set(pool, model.PoolMeta{
		Token0:      "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Token1:      "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Fee:         2500,
		TickSpacing: 60,
	})

	decoder, err := NewV3PoolDecoder(DecoderConfig{})
	if err != nil {
		t.Fatalf("decoder: %v", err)
	}

	ctx := DecodeContext{
		PoolMetaCache: poolMetaCache,
		Logger:        zap.NewNop(),
	}

	sender := common.HexToAddress("0x2222222222222222222222222222222222222222")
	recipient := common.HexToAddress("0x3333333333333333333333333333333333333333")

	data, err := poolABI.Events["Swap"].Inputs.NonIndexed().Pack(
		big.NewInt(-1000),
		big.NewInt(2000),
		big.NewInt(123456789),
		big.NewInt(987654321),
		big.NewInt(-15),
	)
	if err != nil {
		t.Fatalf("pack swap: %v", err)
	}

	logRecord := buildLogRecord(pool, poolABI.Events["Swap"].ID, data, []common.Hash{
		topicFromAddress(sender),
		topicFromAddress(recipient),
	})

	event, err := decoder.Decode(logRecord, ctx)
	if err != nil {
		t.Fatalf("decode swap: %v", err)
	}

	swap, ok := event.Decoded.(model.SwapEventData)
	if !ok {
		t.Fatalf("decoded type mismatch")
	}

	if swap.Amount0 != "-1000" || swap.Amount1 != "2000" {
		t.Fatalf("amounts mismatch: %+v", swap)
	}
	if swap.Tick != -15 {
		t.Fatalf("tick mismatch: %d", swap.Tick)
	}
	if swap.Sender != sender.Hex() || swap.Recipient != recipient.Hex() {
		t.Fatalf("address mismatch")
	}
	if event.PoolMeta.Fee != 2500 || event.PoolMeta.TickSpacing != 60 {
		t.Fatalf("pool meta mismatch")
	}
}

func TestV3PoolDecoderMintBurnCollect(t *testing.T) {
	poolABI, err := V3PoolABI()
	if err != nil {
		t.Fatalf("abi parse: %v", err)
	}

	pool := common.HexToAddress("0x9999999999999999999999999999999999999999")
	poolMetaCache := NewPoolMetaCache()
	poolMetaCache.Set(pool, model.PoolMeta{
		Token0:      "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Token1:      "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Fee:         500,
		TickSpacing: 10,
	})

	decoder, err := NewV3PoolDecoder(DecoderConfig{})
	if err != nil {
		t.Fatalf("decoder: %v", err)
	}

	ctx := DecodeContext{
		PoolMetaCache: poolMetaCache,
		Logger:        zap.NewNop(),
	}

	sender := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	owner := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	recipient := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	mintData, err := poolABI.Events["Mint"].Inputs.NonIndexed().Pack(
		sender,
		big.NewInt(5000),
		big.NewInt(100),
		big.NewInt(200),
	)
	if err != nil {
		t.Fatalf("pack mint: %v", err)
	}

	mintLog := buildLogRecord(pool, poolABI.Events["Mint"].ID, mintData, []common.Hash{
		topicFromAddress(owner),
		topicFromInt24(-120),
		topicFromInt24(120),
	})

	mintEvent, err := decoder.Decode(mintLog, ctx)
	if err != nil {
		t.Fatalf("decode mint: %v", err)
	}

	mint, ok := mintEvent.Decoded.(model.MintEventData)
	if !ok {
		t.Fatalf("mint type mismatch")
	}
	if mint.TickLower != -120 || mint.TickUpper != 120 {
		t.Fatalf("mint tick mismatch: %+v", mint)
	}

	burnData, err := poolABI.Events["Burn"].Inputs.NonIndexed().Pack(
		big.NewInt(7000),
		big.NewInt(300),
		big.NewInt(400),
	)
	if err != nil {
		t.Fatalf("pack burn: %v", err)
	}

	burnLog := buildLogRecord(pool, poolABI.Events["Burn"].ID, burnData, []common.Hash{
		topicFromAddress(owner),
		topicFromInt24(-60),
		topicFromInt24(60),
	})

	burnEvent, err := decoder.Decode(burnLog, ctx)
	if err != nil {
		t.Fatalf("decode burn: %v", err)
	}

	burn, ok := burnEvent.Decoded.(model.BurnEventData)
	if !ok {
		t.Fatalf("burn type mismatch")
	}
	if burn.Amount != "7000" {
		t.Fatalf("burn amount mismatch: %+v", burn)
	}

	collectData, err := poolABI.Events["Collect"].Inputs.NonIndexed().Pack(
		recipient,
		big.NewInt(900),
		big.NewInt(1000),
	)
	if err != nil {
		t.Fatalf("pack collect: %v", err)
	}

	collectLog := buildLogRecord(pool, poolABI.Events["Collect"].ID, collectData, []common.Hash{
		topicFromAddress(owner),
		topicFromInt24(-10),
		topicFromInt24(10),
	})

	collectEvent, err := decoder.Decode(collectLog, ctx)
	if err != nil {
		t.Fatalf("decode collect: %v", err)
	}

	collect, ok := collectEvent.Decoded.(model.CollectEventData)
	if !ok {
		t.Fatalf("collect type mismatch")
	}
	if collect.Amount0 != "900" || collect.Amount1 != "1000" {
		t.Fatalf("collect amount mismatch: %+v", collect)
	}
	if collect.Recipient != recipient.Hex() {
		t.Fatalf("collect recipient mismatch")
	}
}

func buildLogRecord(pool common.Address, topic0 common.Hash, data []byte, indexed []common.Hash) model.LogRecord {
	topics := make([]string, 0, len(indexed)+1)
	topics = append(topics, topic0.Hex())
	for _, topic := range indexed {
		topics = append(topics, topic.Hex())
	}

	return model.LogRecord{
		ChainID:     56,
		BlockNumber: 12345,
		BlockHash:   "0xabc",
		TxHash:      "0xdef",
		LogIndex:    1,
		Address:     pool.Hex(),
		Topics:      topics,
		Data:        hexutil.Encode(data),
		Timestamp:   1700000000,
	}
}

func topicFromAddress(addr common.Address) common.Hash {
	return common.BytesToHash(addr.Bytes())
}

func topicFromInt24(value int32) common.Hash {
	bigVal := big.NewInt(int64(value))
	if value < 0 {
		bigVal = new(big.Int).Add(bigVal, new(big.Int).Lsh(big.NewInt(1), 256))
	}
	return common.BigToHash(bigVal)
}
