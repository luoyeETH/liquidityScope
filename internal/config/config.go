package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds configuration values loaded from flags, env, or config file.
type Config struct {
	RPCURL            string
	FromBlock         uint64
	ToBlock           uint64
	Addresses         []string
	Topic0            []string
	BatchSize         uint64
	Out               string
	Checkpoint        string
	CheckpointEnabled bool
	MaxRetries        int
	RetryBackoff      time.Duration
	LogLevel          string
}

// Load merges config file, environment variables, and flags into Config.
func Load(cfgFile string, flags *pflag.FlagSet) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("INDEXER")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.SetDefault("batch-size", uint64(2000))
	v.SetDefault("out", "./data/logs.jsonl")
	v.SetDefault("checkpoint", "./data/checkpoint.json")
	v.SetDefault("checkpoint-enabled", true)
	v.SetDefault("max-retries", 5)
	v.SetDefault("retry-backoff", 500*time.Millisecond)
	v.SetDefault("log-level", "info")

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return Config{}, fmt.Errorf("bind flags: %w", err)
		}
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return Config{}, fmt.Errorf("read config: %w", err)
			}
		}
	}

	cfg := Config{
		RPCURL:            v.GetString("rpc"),
		FromBlock:         v.GetUint64("from"),
		ToBlock:           v.GetUint64("to"),
		Addresses:         getStringSlice(v, "address"),
		Topic0:            getStringSlice(v, "topic0"),
		BatchSize:         v.GetUint64("batch-size"),
		Out:               v.GetString("out"),
		Checkpoint:        v.GetString("checkpoint"),
		CheckpointEnabled: v.GetBool("checkpoint-enabled"),
		MaxRetries:        v.GetInt("max-retries"),
		RetryBackoff:      v.GetDuration("retry-backoff"),
		LogLevel:          v.GetString("log-level"),
	}

	return cfg, nil
}

func getStringSlice(v *viper.Viper, key string) []string {
	if !v.IsSet(key) {
		return nil
	}

	val := v.Get(key)
	switch typed := val.(type) {
	case []string:
		return cleanStrings(typed)
	case string:
		return splitAndClean(typed)
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return cleanStrings(items)
	default:
		return nil
	}
}

func splitAndClean(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	return cleanStrings(parts)
}

func cleanStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
