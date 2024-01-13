package memory

import (
	"context"
	"time"

	"github.com/savsgio/gotils/strconv"
)

// New returns a new memory provider configured.
func New(cfg Config) (*Provider, error) {
	p := &Provider{
		config: cfg,
	}

	return p, nil
}

// Get returns the data of the given session id.
func (p *Provider) Get(ctx context.Context, id []byte) ([]byte, error) {
	key := p.getKey(id)

	val, found := p.db.Load(key)
	if !found || val == nil { // Not exist
		return nil, nil
	}

	item := val.(*item)

	return item.data, nil
}

// Save saves the session data and expiration from the given session id.
func (p *Provider) Save(ctx context.Context, id, data []byte, expiration time.Duration) error {
	key := p.getKey(id)

	item := acquireItem()
	item.data = data
	item.lastActiveTime = time.Now().UnixNano()
	item.expiration = expiration

	p.db.Store(key, item)

	return nil
}

// Destroy destroys the session from the given id.
func (p *Provider) Destroy(ctx context.Context, id []byte) error {
	key := p.getKey(id)

	return p.destroy(key)
}

func (p *Provider) destroy(key string) error {
	val, found := p.db.LoadAndDelete(key)
	if !found || val == nil {
		return nil
	}

	releaseItem(val.(*item))

	return nil
}

// Regenerate updates the session id and expiration with the new session id
// of the given current session id.
func (p *Provider) Regenerate(ctx context.Context, id, newID []byte, expiration time.Duration) error {
	key := p.getKey(id)

	data, found := p.db.LoadAndDelete(key)
	if found && data != nil {
		item := data.(*item)
		item.lastActiveTime = time.Now().UnixNano()
		item.expiration = expiration

		newKey := p.getKey(newID)

		p.db.Store(newKey, item)
	}

	return nil
}

// Count returns the total of stored sessions.
func (p *Provider) Count(ctx context.Context) (count int) {
	p.db.Range(func(_, _ interface{}) bool {
		count++

		return true
	})

	return count
}

// GC destroys the expired sessions.
func (p *Provider) GC(ctx context.Context) error {
	now := time.Now().UnixNano()

	p.db.Range(func(key, value interface{}) bool {
		item := value.(*item)

		if item.expiration == 0 {
			return true
		}

		if now >= (item.lastActiveTime + item.expiration.Nanoseconds()) {
			_ = p.destroy(key.(string))
		}

		return true
	})

	return nil
}

func (p *Provider) getKey(sessionID []byte) string {
	return strconv.B2S(sessionID)
}
