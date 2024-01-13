package session

import "time"

const (
	defaultSessionKeyName               = "sessionid"
	defaultDomain                       = ""
	defaultExpiration                   = 2 * time.Hour
	defaultGCLifetime                   = 1 * time.Minute
	defaultSecure                       = true
	defaultSessionIDInURLQuery          = false
	defaultSessionIDInHTTPHeader        = false
	defaultCookieLen             uint32 = 32

	expirationAttrKey = "__store:expiration__"

	// If set the cookie expiration when the browser is closed (-1), set the expiration as a keep alive (2 days)
	// so as not to keep dead sessions for a long time
	keepAliveExpiration = 2 * 24 * time.Hour
)
