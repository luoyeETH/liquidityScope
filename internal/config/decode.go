package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// DecodeConfig holds configuration for the decode command.
type DecodeConfig struct {
	RPCURL          string
	In              string
	Out             string
	Errors          string
	LogLevel        string
	Topic0Map       map[string]string
	IncludeLiveMeta bool
}

// LoadDecode merges config file, environment variables, and flags into DecodeConfig.
func LoadDecode(cfgFile string, flags *pflag.FlagSet) (DecodeConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("INDEXER")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	v.SetDefault("out", "./data/typed_events.jsonl")
	v.SetDefault("errors", "./data/decode_errors.jsonl")
	v.SetDefault("include-live-meta", false)
	v.SetDefault("log-level", "info")

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return DecodeConfig{}, fmt.Errorf("bind flags: %w", err)
		}
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return DecodeConfig{}, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return DecodeConfig{}, fmt.Errorf("read config: %w", err)
			}
		}
	}

	cfg := DecodeConfig{
		RPCURL:          v.GetString("rpc"),
		In:              v.GetString("in"),
		Out:             v.GetString("out"),
		Errors:          v.GetString("errors"),
		LogLevel:        v.GetString("log-level"),
		Topic0Map:       getStringMap(v, "topic0-map"),
		IncludeLiveMeta: v.GetBool("include-live-meta"),
	}

	return cfg, nil
}

func getStringMap(v *viper.Viper, key string) map[string]string {
	if !v.IsSet(key) {
		return map[string]string{}
	}

	val := v.Get(key)
	switch typed := val.(type) {
	case map[string]string:
		return typed
	case map[string]interface{}:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = fmt.Sprintf("%v", v)
		}
		return out
	case string:
		return parseStringMap(typed)
	default:
		return map[string]string{}
	}
}

func parseStringMap(input string) map[string]string {
	out := make(map[string]string)
	if strings.TrimSpace(input) == "" {
		return out
	}
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}
