package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// AggregateConfig holds configuration for aggregation.
type AggregateConfig struct {
	RPCURL        string
	Input         string
	Window        string
	PGDSN         string
	BatchSize     int
	StateFile     string
	RecomputeFrom string
	LogLevel      string
}

// LoadAggregate merges config file, environment variables, and flags into AggregateConfig.
func LoadAggregate(cfgFile string, flags *pflag.FlagSet) (AggregateConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("INDEXER")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.SetDefault("batch-size", 1000)
	v.SetDefault("log-level", "info")
	v.SetDefault("window", "5m")

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return AggregateConfig{}, fmt.Errorf("bind flags: %w", err)
		}
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return AggregateConfig{}, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return AggregateConfig{}, fmt.Errorf("read config: %w", err)
			}
		}
	}

	cfg := AggregateConfig{
		RPCURL:        v.GetString("rpc"),
		Input:         v.GetString("in"),
		Window:        v.GetString("window"),
		PGDSN:         v.GetString("pg-dsn"),
		BatchSize:     v.GetInt("batch-size"),
		StateFile:     v.GetString("state-file"),
		RecomputeFrom: v.GetString("recompute-from"),
		LogLevel:      v.GetString("log-level"),
	}

	return cfg, nil
}

// ParseTimestamp parses a timestamp value (unix seconds or RFC3339).
func ParseTimestamp(input string) (uint64, error) {
	if strings.TrimSpace(input) == "" {
		return 0, nil
	}

	if isNumeric(input) {
		val, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}

	tm, err := time.Parse(time.RFC3339, input)
	if err != nil {
		return 0, err
	}
	return uint64(tm.Unix()), nil
}

func isNumeric(input string) bool {
	for _, r := range input {
		if r < '0' || r > '9' {
			return false
		}
	}
	return input != ""
}
