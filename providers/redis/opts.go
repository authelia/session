package redis

import "github.com/redis/go-redis/v9"

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
		provider.keyPrefix = prefix

		return nil
	}
}
