package session

import (
	"github.com/savsgio/gotils/strconv"
	"log"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	defaultLogger Logger = log.New(os.Stderr, "", log.LstdFlags)
)

// New returns a configured manager.
func New(config Config) *Session {
	config.cookieLen = defaultCookieLen

	if config.CookieName == "" {
		config.CookieName = defaultSessionKeyName
	}

	if config.GCLifetime == 0 {
		config.GCLifetime = defaultGCLifetime
	}

	if config.SessionIDInURLQuery && config.SessionNameInURLQuery == "" {
		config.SessionNameInURLQuery = defaultSessionKeyName
	}

	if config.SessionIDInHTTPHeader && config.SessionNameInHTTPHeader == "" {
		config.SessionNameInHTTPHeader = defaultSessionKeyName
	}

	if config.SessionIDGeneratorFunc == nil {
		config.SessionIDGeneratorFunc = config.defaultSessionIDGenerator
	}

	if config.IsSecureFunc == nil {
		config.IsSecureFunc = config.defaultIsSecureFunc
	}

	if config.EncodeFunc == nil {
		config.EncodeFunc = Base64Encode
	}
	if config.DecodeFunc == nil {
		config.DecodeFunc = Base64Decode
	}

	if config.Logger == nil {
		config.Logger = defaultLogger
	}

	session := &Session{
		config: config,
		cookie: newCookie(),
		log:    config.Logger,
		storePool: sync.Pool{
			New: func() interface{} {
				return NewStore()
			},
		},
		stopGCChan: make(chan struct{}),
	}

	return session
}

// SetProvider sets the session provider used by the sessions manager
func (s *Session) SetProvider(provider Provider) error {
	s.provider = provider

	if gcprovider, ok := s.provider.(GarbageCollectedProvider); ok {
		go s.startGC(gcprovider)
	}

	return nil
}

func (s *Session) startGC(provider GarbageCollectedProvider) {
	ticker := time.NewTicker(s.config.GCLifetime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := provider.GC(); err != nil {
				s.log.Printf("session GC crash: %v", err)
			}
		case <-s.stopGCChan:
			return
		}
	}
}

func (s *Session) stopGC() {
	s.stopGCChan <- struct{}{}
}

func (s *Session) setHTTPValues(ctx *fasthttp.RequestCtx, sessionID []byte, expiration time.Duration) {
	secure := s.config.Secure && s.config.IsSecureFunc(ctx)
	s.cookie.set(ctx, s.config.CookieName, sessionID, s.config.Domain, expiration, secure, s.config.CookieSameSite)

	if s.config.SessionIDInHTTPHeader {
		ctx.Request.Header.SetBytesV(s.config.SessionNameInHTTPHeader, sessionID)
		ctx.Response.Header.SetBytesV(s.config.SessionNameInHTTPHeader, sessionID)
	}
}

func (s *Session) delHTTPValues(ctx *fasthttp.RequestCtx) {
	s.cookie.delete(ctx, s.config.CookieName)

	if s.config.SessionIDInHTTPHeader {
		ctx.Request.Header.Del(s.config.SessionNameInHTTPHeader)
		ctx.Response.Header.Del(s.config.SessionNameInHTTPHeader)
	}
}

// Returns the session id from:
// 1. cookie
// 2. http headers
// 3. query string
func (s *Session) getSessionID(ctx *fasthttp.RequestCtx) []byte {
	val := ctx.Request.Header.Cookie(s.config.CookieName)
	if len(val) > 0 {
		return val
	}

	if s.config.SessionIDInHTTPHeader {
		val = ctx.Request.Header.Peek(s.config.SessionNameInHTTPHeader)
		if len(val) > 0 {
			return val
		}
	}

	if s.config.SessionIDInURLQuery {
		val = ctx.FormValue(s.config.SessionNameInURLQuery)
		if len(val) > 0 {
			return val
		}
	}

	return nil
}

// Get returns the user session
// if it does not exist, it will be generated
func (s *Session) Get(ctx *fasthttp.RequestCtx) (*Store, error) {
	if s.provider == nil {
		return nil, ErrNotSetProvider
	}

	newUser := false

	id := s.getSessionID(ctx)
	if len(id) == 0 {
		id = s.config.SessionIDGeneratorFunc()
		if len(id) == 0 {
			return nil, ErrEmptySessionID
		}

		newUser = true
	}

	store := s.storePool.Get().(*Store)
	store.sessionID = id
	store.defaultExpiration = s.config.Expiration

	if !newUser {
		data, err := s.provider.Get(ctx, strconv.B2S(id))
		if err != nil {
			return nil, err
		}

		if err := s.config.DecodeFunc(&store.data, data); err != nil {
			return store, nil
		}
	}

	return store, nil
}

// Save saves the user session
//
// Warning: Don't use the store after exec this function, because, you will lose the after data
// For avoid it, defer this function in your request handler
func (s *Session) Save(ctx *fasthttp.RequestCtx, store *Store) error {
	if s.provider == nil {
		return ErrNotSetProvider
	}

	id := store.GetSessionID()
	lookup := store.GetLookup()
	expiration := store.GetExpiration()

	providerExpiration := expiration
	if expiration == -1 {
		providerExpiration = keepAliveExpiration
	}

	data, err := s.config.EncodeFunc(store.GetAll())
	if err != nil {
		return err
	}

	if err := s.provider.Save(ctx, strconv.B2S(id), lookup, data, providerExpiration); err != nil {
		return err
	}

	s.setHTTPValues(ctx, id, expiration)

	store.Reset()
	s.storePool.Put(store)

	return nil
}

// Regenerate generates a new session id to the current user
func (s *Session) Regenerate(ctx *fasthttp.RequestCtx) error {
	if s.provider == nil {
		return ErrNotSetProvider
	}

	id := s.getSessionID(ctx)
	expiration := s.config.Expiration

	newID := s.config.SessionIDGeneratorFunc()
	if len(newID) == 0 {
		return ErrEmptySessionID
	}

	providerExpiration := expiration
	if expiration == -1 {
		providerExpiration = keepAliveExpiration
	}

	if err := s.provider.Regenerate(ctx, strconv.B2S(id), strconv.B2S(newID), providerExpiration); err != nil {
		return err
	}

	s.setHTTPValues(ctx, newID, expiration)

	return nil
}

// Destroy destroys the session of the current user
func (s *Session) Destroy(ctx *fasthttp.RequestCtx) error {
	if s.provider == nil {
		return ErrNotSetProvider
	}

	id := s.getSessionID(ctx)
	if len(id) == 0 {
		return nil
	}

	err := s.provider.Destroy(ctx, strconv.B2S(id))
	if err != nil {
		return err
	}

	s.delHTTPValues(ctx)

	return nil
}
