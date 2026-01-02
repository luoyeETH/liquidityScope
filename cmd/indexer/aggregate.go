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

	"liquidityScope/internal/aggregate"
	"liquidityScope/internal/chain"
	"liquidityScope/internal/config"
	"liquidityScope/internal/storage/postgres"
)

func runAggregate(cmd *cobra.Command, _ []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadAggregate(cfgFile, cmd.Flags())
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
	if cfg.Input == "" {
		return fmt.Errorf("input path is required")
	}
	if cfg.PGDSN == "" {
		return fmt.Errorf("pg dsn is required")
	}

	windowDuration, err := time.ParseDuration(cfg.Window)
	if err != nil {
		return fmt.Errorf("invalid window: %w", err)
	}
	if windowDuration <= 0 {
		return fmt.Errorf("window must be positive")
	}
	windowSeconds := uint64(windowDuration.Seconds())
	if windowSeconds == 0 {
		return fmt.Errorf("window must be at least 1s")
	}

	recomputeFrom, err := config.ParseTimestamp(cfg.RecomputeFrom)
	if err != nil {
		return fmt.Errorf("parse recompute-from: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chainClient, err := chain.NewClient(ctx, cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("connect rpc: %w", err)
	}
	defer chainClient.Close()

	store, err := postgres.NewStore(ctx, cfg.PGDSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer store.Close()

	var stateStore aggregate.StateStore
	if cfg.StateFile != "" {
		stateStore = &aggregate.FileStateStore{Path: cfg.StateFile}
	} else {
		stateStore = &aggregate.DBStateStore{Store: store, Name: fmt.Sprintf("aggregator:%d", windowSeconds)}
	}

	agg := aggregate.NewAggregator(aggregate.Config{
		WindowSeconds: windowSeconds,
		BatchSize:     cfg.BatchSize,
		RecomputeFrom: recomputeFrom,
		StateStore:    stateStore,
	}, store, chainClient, logger)

	logger.Info("aggregate start",
		zap.String("input", cfg.Input),
		zap.String("pg_dsn", redactDSN(cfg.PGDSN)),
		zap.Uint64("window_seconds", windowSeconds),
		zap.Int("batch_size", cfg.BatchSize),
		zap.Uint64("recompute_from", recomputeFrom),
	)

	return agg.Run(ctx, cfg.Input)
}

func redactDSN(dsn string) string {
	if dsn == "" {
		return dsn
	}
	return "***"
}
