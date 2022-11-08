package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	path := "../testdata/config/bitxhub.toml"
	config, err := UnmarshalConfig("", path)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(config.Addrs))
	assert.Equal(t, 4, len(config.GrpcAddrs))
}
