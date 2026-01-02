package dex

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"liquidityScope/internal/model"
)

// DecoderConfig configures decoder behavior.
type DecoderConfig struct {
	Topic0Map map[string]string
}

// V3PoolDecoder decodes PancakeSwap V3 / Uniswap V3 pool events.
type V3PoolDecoder struct {
	poolABI     abi.ABI
	topicToName map[string]string
}

// NewV3PoolDecoder builds a V3 pool decoder.
func NewV3PoolDecoder(cfg DecoderConfig) (*V3PoolDecoder, error) {
	poolABI, err := V3PoolABI()
	if err != nil {
		return nil, err
	}

	topicToName := map[string]string{
		strings.ToLower(poolABI.Events["Swap"].ID.Hex()):    "Swap",
		strings.ToLower(poolABI.Events["Mint"].ID.Hex()):    "Mint",
		strings.ToLower(poolABI.Events["Burn"].ID.Hex()):    "Burn",
		strings.ToLower(poolABI.Events["Collect"].ID.Hex()): "Collect",
	}

	for topic0, name := range cfg.Topic0Map {
		original := name
		name = normalizeEventName(name)
		if name == "" {
			return nil, fmt.Errorf("unsupported event name in topic0 map: %s", original)
		}
		if topic0 == "" {
			continue
		}
		topicToName[strings.ToLower(topic0)] = name
	}

	return &V3PoolDecoder{
		poolABI:     poolABI,
		topicToName: topicToName,
	}, nil
}

// CanDecode checks if the topic0 is supported.
func (d *V3PoolDecoder) CanDecode(topic0 string) bool {
	if topic0 == "" {
		return false
	}
	_, ok := d.topicToName[strings.ToLower(topic0)]
	return ok
}

// Decode converts a LogRecord into a TypedEvent.
func (d *V3PoolDecoder) Decode(log model.LogRecord, ctx DecodeContext) (*model.TypedEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("missing topics")
	}
	name, ok := d.topicToName[strings.ToLower(log.Topics[0])]
	if !ok {
		return nil, fmt.Errorf("unsupported topic0: %s", log.Topics[0])
	}

	if !common.IsHexAddress(log.Address) {
		return nil, fmt.Errorf("invalid pool address: %s", log.Address)
	}
	pool := common.HexToAddress(log.Address)

	poolMeta, err := getPoolMeta(ctx, pool, log.BlockNumber)
	if err != nil {
		return nil, err
	}

	switch name {
	case "Swap":
		decoded, err := d.decodeSwap(log)
		if err != nil {
			return nil, err
		}
		return buildTypedEvent(log, name, decoded, poolMeta), nil
	case "Mint":
		decoded, err := d.decodeMint(log)
		if err != nil {
			return nil, err
		}
		return buildTypedEvent(log, name, decoded, poolMeta), nil
	case "Burn":
		decoded, err := d.decodeBurn(log)
		if err != nil {
			return nil, err
		}
		return buildTypedEvent(log, name, decoded, poolMeta), nil
	case "Collect":
		decoded, err := d.decodeCollect(log)
		if err != nil {
			return nil, err
		}
		return buildTypedEvent(log, name, decoded, poolMeta), nil
	default:
		return nil, fmt.Errorf("unsupported event name: %s", name)
	}
}

func normalizeEventName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "swap":
		return "Swap"
	case "mint":
		return "Mint"
	case "burn":
		return "Burn"
	case "collect":
		return "Collect"
	default:
		return ""
	}
}

func getPoolMeta(ctx DecodeContext, pool common.Address, blockNumber uint64) (model.PoolMeta, error) {
	var meta model.PoolMeta
	var ok bool
	if ctx.PoolMetaCache != nil {
		meta, ok = ctx.PoolMetaCache.Get(pool)
	}
	if ctx.Chain == nil {
		return model.PoolMeta{}, fmt.Errorf("chain client is nil")
	}

	callCtx := ctx.Context
	if callCtx == nil {
		callCtx = context.Background()
	}

	if !ok {
		var err error
		meta, err = FetchPoolMeta(callCtx, ctx.Chain, pool, ctx.TokenMetaCache, ctx.Logger)
		if err != nil {
			return model.PoolMeta{}, err
		}
		if ctx.PoolMetaCache != nil {
			ctx.PoolMetaCache.Set(pool, meta)
		}
	}

	if ctx.IncludeLiveMeta {
		if optional, err := FetchPoolOptionalMeta(callCtx, ctx.Chain, pool, blockNumber, ctx.Logger); err == nil {
			if optional.Liquidity != "" {
				meta.Liquidity = optional.Liquidity
			}
			if optional.Slot0 != nil {
				meta.Slot0 = optional.Slot0
			}
		}
	}
	return meta, nil
}

func buildTypedEvent(log model.LogRecord, name string, decoded interface{}, meta model.PoolMeta) *model.TypedEvent {
	raw := &model.RawLogRef{Topic0: log.Topics[0], Data: log.Data}
	return &model.TypedEvent{
		ChainID:     log.ChainID,
		BlockNumber: log.BlockNumber,
		BlockHash:   log.BlockHash,
		TxHash:      log.TxHash,
		LogIndex:    log.LogIndex,
		Address:     log.Address,
		EventName:   name,
		Timestamp:   log.Timestamp,
		Decoded:     decoded,
		PoolMeta:    meta,
		Raw:         raw,
	}
}

func (d *V3PoolDecoder) decodeSwap(log model.LogRecord) (model.SwapEventData, error) {
	event := d.poolABI.Events["Swap"]
	indexedTopics, err := parseIndexedTopics(event, log.Topics)
	if err != nil {
		return model.SwapEventData{}, err
	}

	var indexed struct {
		Sender    common.Address
		Recipient common.Address
	}
	if err := abi.ParseTopics(&indexed, indexedArguments(event.Inputs), indexedTopics); err != nil {
		return model.SwapEventData{}, fmt.Errorf("parse topics: %w", err)
	}

	values, err := unpackNonIndexed(event, log.Data)
	if err != nil {
		return model.SwapEventData{}, err
	}
	if len(values) != 5 {
		return model.SwapEventData{}, fmt.Errorf("unexpected swap values: %d", len(values))
	}

	amount0, err := asBigInt(values[0])
	if err != nil {
		return model.SwapEventData{}, err
	}
	amount1, err := asBigInt(values[1])
	if err != nil {
		return model.SwapEventData{}, err
	}
	sqrtPrice, err := asBigInt(values[2])
	if err != nil {
		return model.SwapEventData{}, err
	}
	liquidity, err := asBigInt(values[3])
	if err != nil {
		return model.SwapEventData{}, err
	}
	tickInt, err := asBigInt(values[4])
	if err != nil {
		return model.SwapEventData{}, err
	}
	tick, err := int24FromBig(tickInt)
	if err != nil {
		return model.SwapEventData{}, err
	}

	return model.SwapEventData{
		Sender:       indexed.Sender.Hex(),
		Recipient:    indexed.Recipient.Hex(),
		Amount0:      amount0.String(),
		Amount1:      amount1.String(),
		SqrtPriceX96: sqrtPrice.String(),
		Liquidity:    liquidity.String(),
		Tick:         tick,
	}, nil
}

func (d *V3PoolDecoder) decodeMint(log model.LogRecord) (model.MintEventData, error) {
	event := d.poolABI.Events["Mint"]
	indexedTopics, err := parseIndexedTopics(event, log.Topics)
	if err != nil {
		return model.MintEventData{}, err
	}

	var indexed struct {
		Owner     common.Address
		TickLower *big.Int
		TickUpper *big.Int
	}
	if err := abi.ParseTopics(&indexed, indexedArguments(event.Inputs), indexedTopics); err != nil {
		return model.MintEventData{}, fmt.Errorf("parse topics: %w", err)
	}

	values, err := unpackNonIndexed(event, log.Data)
	if err != nil {
		return model.MintEventData{}, err
	}
	if len(values) != 4 {
		return model.MintEventData{}, fmt.Errorf("unexpected mint values: %d", len(values))
	}

	sender, err := asAddress(values[0])
	if err != nil {
		return model.MintEventData{}, err
	}
	amount, err := asBigInt(values[1])
	if err != nil {
		return model.MintEventData{}, err
	}
	amount0, err := asBigInt(values[2])
	if err != nil {
		return model.MintEventData{}, err
	}
	amount1, err := asBigInt(values[3])
	if err != nil {
		return model.MintEventData{}, err
	}

	tickLower, err := int24FromBig(indexed.TickLower)
	if err != nil {
		return model.MintEventData{}, err
	}
	tickUpper, err := int24FromBig(indexed.TickUpper)
	if err != nil {
		return model.MintEventData{}, err
	}

	return model.MintEventData{
		Sender:    sender.Hex(),
		Owner:     indexed.Owner.Hex(),
		TickLower: tickLower,
		TickUpper: tickUpper,
		Amount:    amount.String(),
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
	}, nil
}

func (d *V3PoolDecoder) decodeBurn(log model.LogRecord) (model.BurnEventData, error) {
	event := d.poolABI.Events["Burn"]
	indexedTopics, err := parseIndexedTopics(event, log.Topics)
	if err != nil {
		return model.BurnEventData{}, err
	}

	var indexed struct {
		Owner     common.Address
		TickLower *big.Int
		TickUpper *big.Int
	}
	if err := abi.ParseTopics(&indexed, indexedArguments(event.Inputs), indexedTopics); err != nil {
		return model.BurnEventData{}, fmt.Errorf("parse topics: %w", err)
	}

	values, err := unpackNonIndexed(event, log.Data)
	if err != nil {
		return model.BurnEventData{}, err
	}
	if len(values) != 3 {
		return model.BurnEventData{}, fmt.Errorf("unexpected burn values: %d", len(values))
	}

	amount, err := asBigInt(values[0])
	if err != nil {
		return model.BurnEventData{}, err
	}
	amount0, err := asBigInt(values[1])
	if err != nil {
		return model.BurnEventData{}, err
	}
	amount1, err := asBigInt(values[2])
	if err != nil {
		return model.BurnEventData{}, err
	}

	tickLower, err := int24FromBig(indexed.TickLower)
	if err != nil {
		return model.BurnEventData{}, err
	}
	tickUpper, err := int24FromBig(indexed.TickUpper)
	if err != nil {
		return model.BurnEventData{}, err
	}

	return model.BurnEventData{
		Owner:     indexed.Owner.Hex(),
		TickLower: tickLower,
		TickUpper: tickUpper,
		Amount:    amount.String(),
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
	}, nil
}

func (d *V3PoolDecoder) decodeCollect(log model.LogRecord) (model.CollectEventData, error) {
	event := d.poolABI.Events["Collect"]
	indexedTopics, err := parseIndexedTopics(event, log.Topics)
	if err != nil {
		return model.CollectEventData{}, err
	}

	var indexed struct {
		Owner     common.Address
		TickLower *big.Int
		TickUpper *big.Int
	}
	if err := abi.ParseTopics(&indexed, indexedArguments(event.Inputs), indexedTopics); err != nil {
		return model.CollectEventData{}, fmt.Errorf("parse topics: %w", err)
	}

	values, err := unpackNonIndexed(event, log.Data)
	if err != nil {
		return model.CollectEventData{}, err
	}
	if len(values) != 3 {
		return model.CollectEventData{}, fmt.Errorf("unexpected collect values: %d", len(values))
	}

	recipient, err := asAddress(values[0])
	if err != nil {
		return model.CollectEventData{}, err
	}
	amount0, err := asBigInt(values[1])
	if err != nil {
		return model.CollectEventData{}, err
	}
	amount1, err := asBigInt(values[2])
	if err != nil {
		return model.CollectEventData{}, err
	}

	tickLower, err := int24FromBig(indexed.TickLower)
	if err != nil {
		return model.CollectEventData{}, err
	}
	tickUpper, err := int24FromBig(indexed.TickUpper)
	if err != nil {
		return model.CollectEventData{}, err
	}

	return model.CollectEventData{
		Owner:     indexed.Owner.Hex(),
		Recipient: recipient.Hex(),
		TickLower: tickLower,
		TickUpper: tickUpper,
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
	}, nil
}

func parseIndexedTopics(event abi.Event, topics []string) ([]common.Hash, error) {
	indexedCount := len(indexedArguments(event.Inputs))
	if len(topics) != indexedCount+1 {
		return nil, fmt.Errorf("expected %d topics, got %d", indexedCount+1, len(topics))
	}
	return parseTopicHashes(topics[1:])
}

func parseTopicHashes(topics []string) ([]common.Hash, error) {
	out := make([]common.Hash, 0, len(topics))
	for _, topic := range topics {
		data, err := hexutil.Decode(topic)
		if err != nil {
			return nil, fmt.Errorf("invalid topic: %w", err)
		}
		if len(data) > 32 {
			return nil, fmt.Errorf("topic length %d", len(data))
		}
		out = append(out, common.BytesToHash(data))
	}
	return out, nil
}

func indexedArguments(args abi.Arguments) abi.Arguments {
	indexed := make(abi.Arguments, 0, len(args))
	for _, arg := range args {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return indexed
}

func unpackNonIndexed(event abi.Event, dataHex string) ([]interface{}, error) {
	data, err := hexutil.Decode(dataHex)
	if err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}
	values, err := event.Inputs.NonIndexed().Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", event.Name, err)
	}
	return values, nil
}
