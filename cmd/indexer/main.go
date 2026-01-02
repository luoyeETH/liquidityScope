package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/config"
	"liquidityScope/internal/indexer"
	"liquidityScope/internal/storage"
)

func main() {
	root := &cobra.Command{
		Use:          "indexer",
		Short:        "BSC log indexer",
		SilenceUsage: true,
	}

	root.PersistentFlags().String("config", "", "config file path")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the indexer",
		RunE:  runIndexer,
	}

	runCmd.Flags().String("rpc", "", "BSC RPC URL")
	runCmd.Flags().Uint64("from", 0, "start block (inclusive)")
	runCmd.Flags().Uint64("to", 0, "end block (inclusive), 0 means latest")
	runCmd.Flags().StringSlice("address", nil, "contract addresses (comma-separated)")
	runCmd.Flags().StringSlice("topic0", nil, "topic0 signatures (comma-separated)")
	runCmd.Flags().Uint64("batch-size", 2000, "blocks per batch")
	runCmd.Flags().String("out", "./data/logs.jsonl", "output JSONL path")
	runCmd.Flags().String("checkpoint", "./data/checkpoint.json", "checkpoint file path")
	runCmd.Flags().Bool("checkpoint-enabled", true, "enable checkpointing")
	runCmd.Flags().Int("max-retries", 5, "maximum retry attempts")
	runCmd.Flags().Duration("retry-backoff", 500*time.Millisecond, "initial retry backoff")
	runCmd.Flags().String("log-level", "info", "log level (debug, info, warn, error)")

	root.AddCommand(runCmd)

	decodeCmd := &cobra.Command{
		Use:   "decode",
		Short: "Decode raw logs into typed events",
		RunE:  runDecode,
	}

	decodeCmd.Flags().String("rpc", "", "BSC RPC URL")
	decodeCmd.Flags().String("in", "", "input raw logs JSONL")
	decodeCmd.Flags().String("out", "./data/typed_events.jsonl", "output typed events JSONL")
	decodeCmd.Flags().String("errors", "./data/decode_errors.jsonl", "decode errors JSONL")
	decodeCmd.Flags().String("topic0-map", "", "extra topic0->event mappings (comma-separated key=value)")
	decodeCmd.Flags().Bool("include-live-meta", false, "include optional slot0/liquidity (requires archive RPC for historical accuracy)")
	decodeCmd.Flags().String("log-level", "info", "log level (debug, info, warn, error)")

	root.AddCommand(decodeCmd)

	aggregateCmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Aggregate typed events into window metrics",
		RunE:  runAggregate,
	}

	aggregateCmd.Flags().String("rpc", "", "BSC RPC URL")
	aggregateCmd.Flags().String("in", "", "input typed events JSONL")
	aggregateCmd.Flags().String("window", "5m", "aggregation window (e.g. 1m, 5m, 1h)")
	aggregateCmd.Flags().String("pg-dsn", "", "Postgres DSN")
	aggregateCmd.Flags().Int("batch-size", 1000, "batch size for DB writes")
	aggregateCmd.Flags().String("state-file", "", "optional local state file for progress tracking")
	aggregateCmd.Flags().String("recompute-from", "", "recompute from timestamp (unix seconds or RFC3339)")
	aggregateCmd.Flags().String("log-level", "info", "log level (debug, info, warn, error)")

	root.AddCommand(aggregateCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runIndexer(cmd *cobra.Command, _ []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(cfgFile, cmd.Flags())
	if err != nil {
		return err
	}

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	if cfg.RPCURL == "" {
		return fmt.Errorf("rpc url is required")
	}

	addresses, err := indexer.ParseAddresses(cfg.Addresses)
	if err != nil {
		return err
	}
	if len(addresses) == 0 {
		return fmt.Errorf("address list is required")
	}

	topic0, err := indexer.ParseTopic0(cfg.Topic0)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chainClient, err := chain.NewClient(ctx, cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("connect rpc: %w", err)
	}
	defer chainClient.Close()

	storageSink := storage.NewJsonlStorage(cfg.Out)

	runner := indexer.NewRunner(indexer.RunConfig{
		FromBlock:         cfg.FromBlock,
		ToBlock:           cfg.ToBlock,
		Addresses:         addresses,
		Topic0:            topic0,
		BatchSize:         cfg.BatchSize,
		CheckpointPath:    cfg.Checkpoint,
		CheckpointEnabled: cfg.CheckpointEnabled,
		MaxRetries:        cfg.MaxRetries,
		RetryBackoff:      cfg.RetryBackoff,
	}, chainClient, storageSink, logger)

	logger.Info("indexer start",
		zap.String("rpc", cfg.RPCURL),
		zap.Uint64("from", cfg.FromBlock),
		zap.Uint64("to", cfg.ToBlock),
		zap.Int("addresses", len(addresses)),
		zap.Int("topic0", len(topic0)),
		zap.Uint64("batch_size", cfg.BatchSize),
		zap.String("out", cfg.Out),
		zap.Bool("checkpoint_enabled", cfg.CheckpointEnabled),
		zap.String("checkpoint", cfg.Checkpoint),
	)

	return runner.Run(ctx)
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevel()
	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}

	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}
