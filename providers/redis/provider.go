package redis

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/bytebufferpool"
)

func New(ctx context.Context, db redis.Cmdable, opts ...Option) (provider *Provider, err error) {
	provider = &Provider{keyPrefix: "gopher", seps: 2, db: db}

	for _, opt := range opts {
		if err = opt(provider); err != nil {
			return nil, err
		}
	}

	if err = provider.db.Ping(ctx).Err(); err != nil {
		return nil, newErrRedisConnection(err)
	}

	return provider, nil
}

// NewStandalone returns a new configured redis standalone provider.
func NewStandalone(ctx context.Context, config ConfigStandalone) (provider *Provider, err error) {
	if config.Addr == "" {
		return nil, ErrConfigAddrEmpty
	}

	db := redis.NewClient(&redis.Options{
		Network:               config.Network,
		Addr:                  config.Addr,
		ClientName:            config.ClientName,
		Dialer:                config.Dialer,
		OnConnect:             config.OnConnect,
		Protocol:              config.Protocol,
		Username:              config.Username,
		Password:              config.Password,
		CredentialsProvider:   config.CredentialsProvider,
		DB:                    config.DB,
		MaxRetries:            config.MaxRetries,
		MinRetryBackoff:       config.MinRetryBackoff,
		MaxRetryBackoff:       config.MaxRetryBackoff,
		DialTimeout:           config.DialTimeout,
		ReadTimeout:           config.ReadTimeout,
		WriteTimeout:          config.WriteTimeout,
		ContextTimeoutEnabled: config.ContextTimeoutEnabled,
		PoolFIFO:              config.PoolFIFO,
		PoolSize:              config.PoolSize,
		PoolTimeout:           config.PoolTimeout,
		MinIdleConns:          config.MinIdleConns,
		MaxIdleConns:          config.MaxIdleConns,
		MaxActiveConns:        config.MaxActiveConns,
		ConnMaxIdleTime:       config.ConnMaxIdleTime,
		ConnMaxLifetime:       config.ConnMaxLifetime,
		TLSConfig:             config.TLSConfig,
		Limiter:               config.Limiter,
		DisableIndentity:      config.DisableIdentity,
		IdentitySuffix:        config.IdentitySuffix,
	})

	return New(ctx, db, WithLogger(config.Logger), WithKeyPrefix(config.KeyPrefix))
}

// NewSentinel returns a new redis provider using sentinel to determine the redis server to connect to.
func NewSentinel(ctx context.Context, config ConfigSentinel) (provider *Provider, err error) {
	if config.MasterName == "" {
		return nil, ErrConfigMasterNameEmpty
	}

	db := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:              config.MasterName,
		SentinelAddrs:           config.SentinelAddrs,
		ClientName:              config.ClientName,
		SentinelUsername:        config.SentinelUsername,
		SentinelPassword:        config.SentinelPassword,
		RouteByLatency:          config.RouteByLatency,
		RouteRandomly:           config.RouteRandomly,
		ReplicaOnly:             config.ReplicaOnly,
		UseDisconnectedReplicas: config.UseDisconnectedReplicas,
		Dialer:                  config.Dialer,
		OnConnect:               config.OnConnect,
		Protocol:                config.Protocol,
		Username:                config.Username,
		Password:                config.Password,
		DB:                      config.DB,
		MaxRetries:              config.MaxRetries,
		MinRetryBackoff:         config.MinRetryBackoff,
		MaxRetryBackoff:         config.MaxRetryBackoff,
		DialTimeout:             config.DialTimeout,
		ReadTimeout:             config.ReadTimeout,
		WriteTimeout:            config.WriteTimeout,
		ContextTimeoutEnabled:   config.ContextTimeoutEnabled,
		PoolFIFO:                config.PoolFIFO,
		PoolSize:                config.PoolSize,
		PoolTimeout:             config.PoolTimeout,
		MinIdleConns:            config.MinIdleConns,
		MaxIdleConns:            config.MaxIdleConns,
		MaxActiveConns:          config.MaxActiveConns,
		ConnMaxIdleTime:         config.ConnMaxIdleTime,
		ConnMaxLifetime:         config.ConnMaxLifetime,
		TLSConfig:               config.TLSConfig,
		DisableIndentity:        config.DisableIdentity,
		IdentitySuffix:          config.IdentitySuffix,
	})

	return New(ctx, db, WithLogger(config.Logger), WithKeyPrefix(config.KeyPrefix))
}

// Get returns the data of the given session id.
func (p *Provider) Get(ctx context.Context, id string) (data []byte, err error) {
	key := p.getSessionIDKey(id)

	reply, err := p.db.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	return reply, nil

}

// Lookup retrieves the session ID's for a given lookup value.
func (p *Provider) Lookup(ctx context.Context, lookup string) (ids []string, err error) {
	keys, err := p.db.Keys(ctx, p.getLookupKeyWildcard(lookup)).Result()
	if err != nil {
		return nil, err
	}

	n := len(keys)

	ids = make([]string, n)

	for i := 0; i < n; i++ {
		ids[i] = keys[i][strings.LastIndex(keys[i], keySep)+1:]
	}

	return ids, nil
}

// Save saves the session data and expiration from the given session id.
func (p *Provider) Save(ctx context.Context, id, lookup string, data []byte, expiration time.Duration) (err error) {
	if err = p.db.Set(ctx, p.getSessionIDKey(id), data, expiration).Err(); err != nil {
		return err
	}

	if len(lookup) == 0 {
		return nil
	}

	return p.db.Set(ctx, p.getLookupKey(id, lookup), 1, expiration).Err()
}

// Destroy destroys the session from the given id.
func (p *Provider) Destroy(ctx context.Context, id string) error {
	return p.db.Del(ctx, p.getSessionIDKey(id)).Err()
}

// Regenerate updates the session id and expiration with the new session id
// of the given current session id.
func (p *Provider) Regenerate(ctx context.Context, id, newID string, expiration time.Duration) (err error) {
	key := p.getSessionIDKey(id)
	newKey := p.getSessionIDKey(newID)

	exists, err := p.db.Exists(ctx, key).Result()
	if err != nil {
		return err
	}

	if exists > 0 { // Exist
		if err = p.db.Rename(ctx, key, newKey).Err(); err != nil {
			return err
		}

		if err = p.db.Expire(ctx, newKey, expiration).Err(); err != nil {
			return err
		}
	}

	return nil
}

// Count returns the total of stored sessions
func (p *Provider) Count(ctx context.Context) int {
	reply, err := p.db.Keys(ctx, p.getSessionIDKey(keyWildcard)).Result()
	if err != nil {
		return 0
	}

	return len(reply)
}

func (p *Provider) getSessionIDKey(sessionID string) string {
	key := bytebufferpool.Get()

	key.SetString(p.keyPrefix)
	_, _ = key.WriteString(keySepSession)
	_, _ = key.WriteString(sessionID)

	keyStr := key.String()

	bytebufferpool.Put(key)

	return keyStr
}

func (p *Provider) getLookupKey(sessionID, lookup string) string {
	key := bytebufferpool.Get()

	key.SetString(p.keyPrefix)
	_, _ = key.WriteString(keySepLookup)
	_, _ = key.WriteString(lookup)
	_, _ = key.WriteString(keySep)
	_, _ = key.WriteString(sessionID)

	keyStr := key.String()

	bytebufferpool.Put(key)

	return keyStr
}

func (p *Provider) getLookupKeyDelete(sessionID, lookup string) string {
	key := bytebufferpool.Get()

	key.SetString(p.keyPrefix)
	_, _ = key.WriteString(keySepLookup)
	_, _ = key.WriteString(lookup)
	_, _ = key.WriteString(keySep)
	_, _ = key.WriteString(sessionID)

	keyStr := key.String()

	bytebufferpool.Put(key)

	return keyStr
}

func (p *Provider) getLookupKeyWildcard(lookup string) string {
	key := bytebufferpool.Get()

	key.SetString(p.keyPrefix)
	_, _ = key.WriteString(keySepLookup)
	_, _ = key.WriteString(lookup)
	_, _ = key.WriteString(keySepWildcard)

	keyStr := key.String()

	bytebufferpool.Put(key)

	return keyStr
}
