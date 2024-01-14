package memory

import (
	"context"
	"time"
)

// New returns a new memory provider configured.
func New(config Config) (*Provider, error) {
	p := &Provider{
		config: config,
	}

	return p, nil
}

// Get returns the data of the given session id.
func (p *Provider) Get(ctx context.Context, id string) (data []byte, err error) {
	val, found := p.db.Load(id)
	if !found || val == nil { // Not exist
		return nil, nil
	}

	item := val.(*databaseItem)

	return item.data, nil
}

// Lookup returns the IDs for the given lookup.
func (p *Provider) Lookup(ctx context.Context, lookup string) (ids []string, err error) {
	p.db.Range(func(key, value any) bool {
		if item, ok := value.(*databaseItem); ok && item.lookup == lookup {
			ids = append(ids, key.(string))
		}

		return true
	})

	return ids, nil
}

// Save saves the session data and expiration from the given session id.
func (p *Provider) Save(ctx context.Context, id, lookup string, data []byte, expiration time.Duration) (err error) {
	item := acquireDatabaseItem()
	item.data = data
	item.lookup = lookup
	item.lastActiveTime = time.Now().UnixNano()
	item.expiration = expiration

	p.db.Store(id, item)

	return nil
}

// Destroy destroys the session from the given id.
func (p *Provider) Destroy(ctx context.Context, id string) (err error) {
	return p.destroy(id)
}

func (p *Provider) destroy(key string) (err error) {
	val, found := p.db.LoadAndDelete(key)
	if !found || val == nil {
		return nil
	}

	item := val.(*databaseItem)

	if len(item.lookup) != 0 {

	}

	releaseDatabaseItem(item)

	return nil
}

// Regenerate updates the session id and expiration with the new session id
// of the given current session id.
func (p *Provider) Regenerate(ctx context.Context, id, newID string, expiration time.Duration) (err error) {
	data, found := p.db.LoadAndDelete(id)
	if found && data != nil {
		item := data.(*databaseItem)
		item.lastActiveTime = time.Now().UnixNano()
		item.expiration = expiration

		p.db.Store(newID, item)
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
func (p *Provider) GC(ctx context.Context) (err error) {
	now := time.Now().UnixNano()

	p.db.Range(func(key, value interface{}) bool {
		item := value.(*databaseItem)

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
