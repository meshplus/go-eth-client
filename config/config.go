package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const configName = "bitxhub.toml"

type JsonRpc struct {
	Addrs     []string `mapstructure:"http_addrs" toml:"http_addrs" json:"http_addrs"`
	GrpcAddrs []string `mapstructure:"grpc_addrs" toml:"grpc_addrs" json:"grpc_addrs"`
}

type Config struct {
	JsonRpc `mapstructure:"json_rpc" toml:"json_rpc" json:"json_rpc"`
}

func DefaultConfig() *Config {
	return &Config{
		JsonRpc{
			Addrs: []string{"http://localhost:8881", "http://localhost:8882", "http://localhost:8883", "http://localhost:8884"},
		},
	}
}

func UnmarshalConfig(repoPath, configPath string) (*Config, error) {
	v := viper.New()
	if len(configPath) == 0 {
		viper.SetConfigFile(filepath.Join(repoPath, configName))
	} else {
		v.SetConfigFile(configPath)
		fileData, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read bitxhub config error: %w", err)
		}
		err = ioutil.WriteFile(filepath.Join(repoPath, configName), fileData, 0644)
		if err != nil {
			return nil, fmt.Errorf("write bitxhub config failed: %w", err)
		}
	}
	v.SetConfigType("toml")
	v.AutomaticEnv()
	v.SetEnvPrefix("BITXHUB")
	replacer := strings.NewReplacer(".", "_")
	v.SetEnvKeyReplacer(replacer)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("readInConfig error: %w", err)
	}

	//config := DefaultConfig()
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unmarshal config error: %w", err)
	}

	// reading configuration will not cover the default configuration when type is string slice
	if len(config.Addrs) == 0 {
		config.Addrs = []string{"http://localhost:8881", "http://localhost:8882", "http://localhost:8883", "http://localhost:8884"}
	}

	if len(config.GrpcAddrs) == 0 {
		config.GrpcAddrs = []string{"localhost:60011"}
	}
	return config, nil
}
