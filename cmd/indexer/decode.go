package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/config"
	"liquidityScope/internal/dex"
	"liquidityScope/internal/model"
)

func runDecode(cmd *cobra.Command, _ []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadDecode(cfgFile, cmd.Flags())
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
	if cfg.In == "" {
		return fmt.Errorf("input path is required")
	}
	if cfg.Out == "" {
		return fmt.Errorf("output path is required")
	}
	if cfg.Errors == "" {
		return fmt.Errorf("errors path is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chainClient, err := chain.NewClient(ctx, cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("connect rpc: %w", err)
	}
	defer chainClient.Close()

	decoder, err := dex.NewV3PoolDecoder(dex.DecoderConfig{Topic0Map: cfg.Topic0Map})
	if err != nil {
		return err
	}

	decodeCtx := dex.DecodeContext{
		Context:         ctx,
		Chain:           chainClient,
		PoolMetaCache:   dex.NewPoolMetaCache(),
		TokenMetaCache:  dex.NewTokenMetaCache(),
		Logger:          logger,
		IncludeLiveMeta: cfg.IncludeLiveMeta,
	}

	inputFile, err := os.Open(cfg.In)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer inputFile.Close()

	outWriter, err := newJSONLWriter(cfg.Out, false)
	if err != nil {
		return err
	}
	defer outWriter.Close()

	errWriter, err := newJSONLWriter(cfg.Errors, false)
	if err != nil {
		return err
	}
	defer errWriter.Close()

	logger.Info("decode start",
		zap.String("rpc", cfg.RPCURL),
		zap.String("in", cfg.In),
		zap.String("out", cfg.Out),
		zap.String("errors", cfg.Errors),
		zap.Bool("include_live_meta", cfg.IncludeLiveMeta),
	)

	scanner := bufio.NewScanner(inputFile)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var total, decoded, skipped, failed int
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		total++

		var record model.LogRecord
		if err := json.Unmarshal(line, &record); err != nil {
			failed++
			writeDecodeError(errWriter, model.DecodeError{Error: err.Error()})
			continue
		}
		if len(record.Topics) == 0 {
			failed++
			writeDecodeError(errWriter, decodeErrorFromRecord(record, fmt.Errorf("missing topic0")))
			continue
		}

		if !decoder.CanDecode(record.Topics[0]) {
			skipped++
			continue
		}

		event, err := decoder.Decode(record, decodeCtx)
		if err != nil {
			failed++
			writeDecodeError(errWriter, decodeErrorFromRecord(record, err))
			continue
		}

		if err := outWriter.Write(event); err != nil {
			return err
		}
		decoded++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan input: %w", err)
	}

	logger.Info("decode complete",
		zap.Int("total", total),
		zap.Int("decoded", decoded),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
	)

	return nil
}

type jsonlWriter struct {
	file   *os.File
	writer *bufio.Writer
}

func newJSONLWriter(path string, appendMode bool) (*jsonlWriter, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create dir: %w", err)
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return &jsonlWriter{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (w *jsonlWriter) Write(value interface{}) error {
	line, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := w.writer.Write(line); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return nil
}

func (w *jsonlWriter) Close() error {
	if w == nil {
		return nil
	}
	if err := w.writer.Flush(); err != nil {
		w.file.Close()
		return err
	}
	return w.file.Close()
}

func decodeErrorFromRecord(record model.LogRecord, err error) model.DecodeError {
	topic0 := ""
	if len(record.Topics) > 0 {
		topic0 = record.Topics[0]
	}

	return model.DecodeError{
		ChainID:     record.ChainID,
		BlockNumber: record.BlockNumber,
		TxHash:      record.TxHash,
		LogIndex:    record.LogIndex,
		Address:     record.Address,
		Topic0:      topic0,
		Error:       err.Error(),
	}
}

func writeDecodeError(writer *jsonlWriter, errRecord model.DecodeError) {
	if writer == nil {
		return
	}
	_ = writer.Write(errRecord)
}
