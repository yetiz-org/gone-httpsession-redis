// Package redis provides a Redis-based session provider implementation
// for the gone HTTP session management system.
package redis

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/yetiz-org/gone/ghttp/httpsession"
	datastore "github.com/yetiz-org/goth-datastore"
)

var (
	// SessionPrefix defines the default prefix for Redis session keys
	SessionPrefix = "httpsession"
)

// SessionTypeRedis defines the session type identifier for Redis sessions
const SessionTypeRedis httpsession.SessionType = "REDIS"

// sessionPrefix returns the complete prefix for Redis session keys
func sessionPrefix() string {
	return SessionPrefix + ":gts:"
}

// sessionKey generates a complete Redis key for the given session ID
func sessionKey(sessionId string) string {
	return sessionPrefix() + sessionId
}

// SessionProvider implements the httpsession.SessionProvider interface
// using Redis as the backend storage. It supports both master and slave
// Redis instances for read/write separation.
type SessionProvider struct {
	Master datastore.RedisOperator // Master Redis instance for write operations
	Slave  datastore.RedisOperator // Slave Redis instance for read operations
}

// Type returns the session type identifier for this provider
func (s *SessionProvider) Type() httpsession.SessionType {
	return SessionTypeRedis
}

// NewSession creates a new session with the specified expiration time
func (s *SessionProvider) NewSession(expire time.Time) httpsession.Session {
	session := httpsession.NewDefaultSession(s)
	session.SetExpire(expire)
	return session
}

// Sessions retrieves all active sessions from Redis storage.
// It prefers the slave instance for read operations, falling back to master if slave is unavailable.
// Returns an empty map if no Redis instance is available.
func (s *SessionProvider) Sessions() map[string]httpsession.Session {
	sessions := make(map[string]httpsession.Session)
	op := s.Slave
	if op == nil {
		op = s.Master
	}

	if op == nil {
		return sessions
	}

	keys := op.Keys(sessionPrefix() + "*")
	for _, entity := range keys.GetSlice() {
		if entity.GetString() != "" {
			session := &httpsession.DefaultSession{}
			if err := json.Unmarshal([]byte(entity.GetString()), session); err == nil {
				sessions[session.Id()] = session
			}
		}
	}

	return sessions
}

// Session retrieves a specific session by its key from Redis storage.
// It prefers the slave instance for read operations, falling back to master if slave is unavailable.
// Returns nil if the session is not found or if no Redis instance is available.
func (s *SessionProvider) Session(key string) httpsession.Session {
	op := s.Slave
	if op == nil {
		op = s.Master
	}

	if op == nil {
		return nil
	}

	entity := op.Get(sessionKey(key))
	if entity.Error == nil {
		session := httpsession.NewDefaultSession(s)
		if err := json.Unmarshal([]byte(entity.GetString()), session); err == nil {
			return session
		}
	}

	return nil
}

// Save persists a session to Redis storage with expiration time.
// It prefers the master instance for write operations, falling back to slave if master is unavailable.
// Returns an error if no Redis instance is available, session is nil, or session has expired.
func (s *SessionProvider) Save(session httpsession.Session) error {
	op := s.Master
	if op == nil {
		op = s.Slave
	}

	if op == nil {
		return errors.Errorf("provider is nil")
	}

	if session == nil {
		return errors.Errorf("session is nil")
	}

	if session.IsExpire() {
		return errors.Errorf("session expired")
	}

	ttl := int64(time.Until(session.Expire()).Seconds())
	sessionData, _ := json.Marshal(session)
	if err := op.SetExpire(sessionKey(session.Id()), string(sessionData), ttl).Error; err != nil {
		return err
	}

	return nil
}

// Delete removes a session from Redis storage by its key.
// It prefers the master instance for write operations, falling back to slave if master is unavailable.
// No action is taken if no Redis instance is available.
func (s *SessionProvider) Delete(key string) {
	op := s.Master
	if op == nil {
		op = s.Slave
	}

	if op == nil {
		return
	}

	op.Delete(sessionKey(key))
}

// NewSessionProvider creates a new SessionProvider using a Redis profile name.
// It initializes both master and slave Redis connections based on the profile configuration.
func NewSessionProvider(profileName string) *SessionProvider {
	provider := &SessionProvider{}
	redis := datastore.NewRedis(profileName)
	provider.Master = redis.Master()
	provider.Slave = redis.Slave()
	return provider
}

// NewSessionProviderWithRedis creates a new SessionProvider using an existing Redis instance.
// It extracts both master and slave connections from the provided Redis instance.
func NewSessionProviderWithRedis(redis *datastore.Redis) *SessionProvider {
	provider := &SessionProvider{}
	provider.Master = redis.Master()
	provider.Slave = redis.Slave()
	return provider
}
