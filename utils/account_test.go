package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	accountKey = "account.key"
)

func TestGenAndStoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "account")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(tmpDir))
	}()
	_, addr, err := GenAndStoreAccount(tmpDir, "", accountKey)
	assert.Nil(t, err)
	fmt.Println(addr)
}

func TestNewAccount(t *testing.T) {
	priv, addr, err := NewAccount()
	assert.Nil(t, err)
	assert.NotEmpty(t, priv)
	addr1 := GetPrivateKeyAddr(priv)
	assert.NotEmpty(t, addr1)
	assert.Equal(t, addr, addr1.String())
}
