package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	DefaultPassword = "bitxhub"
	accountKey      = "account.key"
)

func TestNewAccount(t *testing.T) {
	//tmpDir := os.TempDir()
	//defer os.Remove(tmpDir)
	_, addr, err := NewAccount("./", "", accountKey)
	assert.Nil(t, err)
	fmt.Println(addr)
}
