package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type refreshRec struct {
	user UserRecord
	exp  time.Time
}

type memoryRefreshStore struct {
	mu   sync.Mutex
	byID map[string]refreshRec
}

func (m *memoryRefreshStore) ensureMap() {
	if m.byID == nil {
		m.byID = make(map[string]refreshRec)
	}
}

func (m *memoryRefreshStore) newToken(user UserRecord) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMap()
	m.byID[token] = refreshRec{
		user: user,
		exp:  time.Now().Add(refreshTTL()),
	}
	return token, nil
}

func (m *memoryRefreshStore) consume(token string) (UserRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMap()
	rec, ok := m.byID[token]
	if !ok || time.Now().After(rec.exp) {
		if ok {
			delete(m.byID, token)
		}
		return UserRecord{}, false
	}
	delete(m.byID, token)
	return rec.user, true
}

func (m *memoryRefreshStore) revoke(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMap()
	delete(m.byID, token)
}
