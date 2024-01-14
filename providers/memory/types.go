package memory

import (
	"sync"
)

// Config provider settings
type Config struct{}

// Provider backend manager
type Provider struct {
	config Config
	db     sync.Map
	lookup sync.Map
}
