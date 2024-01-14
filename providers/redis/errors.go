package redis

import (
	"errors"
	"fmt"
)

var (
	ErrConfigAddrEmpty       = errors.New("Config Addr must not be empty")
	ErrConfigMasterNameEmpty = errors.New("Config MasterName must not be empty")
	ErrEmptyPrefix           = errors.New("prefix must have a value but it is empty")
)

func newErrRedisConnection(err error) error {
	return fmt.Errorf("redis connection error: %w", err)
}
