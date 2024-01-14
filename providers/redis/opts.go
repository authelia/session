package redis

import (
	"github.com/redis/go-redis/v9"
	"strings"
)

type Option func(provider *Provider) (err error)

func WithLogger(logger Logger) Option {
	return func(provider *Provider) (err error) {
		if logger == nil {
			return nil
		}

		redis.SetLogger(logger)

		return nil
	}
}

func WithKeyPrefix(prefix string) Option {
	return func(provider *Provider) (err error) {
		if len(prefix) == 0 {
			return ErrEmptyPrefix
		}

		provider.keyPrefix = prefix
		provider.seps = strings.Count(prefix, keySep) + 2

		return nil
	}
}
