package session

import (
	"errors"
)

var (
	ErrNotSetProvider = errors.New("session: provider not set or configured")
	ErrEmptySessionID = errors.New("empty session id")
)
