package redis

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/yetiz-org/gone/ghttp/httpsession"
	datastore "github.com/yetiz-org/goth-datastore"
)

var (
	SessionPrefix = "httpsession"
)

const SessionTypeRedis httpsession.SessionType = "REDIS"

func sessionPrefix() string {
	return SessionPrefix + ":gts:"
}

func sessionKey(sessionId string) string {
	return sessionPrefix() + sessionId
}

type SessionProvider struct {
	Master *datastore.RedisOp
	Slave  *datastore.RedisOp
}

func (s *SessionProvider) Type() httpsession.SessionType {
	return SessionTypeRedis
}

func (s *SessionProvider) NewSession(expire time.Time) httpsession.Session {
	session := httpsession.NewDefaultSession(s)
	session.SetExpire(expire)
	return session
}

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
		session := &httpsession.DefaultSession{}
		if err := json.Unmarshal([]byte(entity.GetString()), session); err == nil {
			return session
		}
	}

	return nil
}

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

func NewSessionProvider(profileName string) *SessionProvider {
	provider := &SessionProvider{}
	redis := datastore.NewRedis(profileName)
	provider.Master = redis.Master()
	provider.Slave = redis.Slave()
	return provider
}

func NewSessionProviderWithRedis(redis *datastore.Redis) *SessionProvider {
	provider := &SessionProvider{}
	provider.Master = redis.Master()
	provider.Slave = redis.Slave()
	return provider
}
