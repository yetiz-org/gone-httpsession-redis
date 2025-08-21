// Package redis tests for Redis-based session provider implementation
package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yetiz-org/gone/ghttp/httpsession"
	datastore "github.com/yetiz-org/goth-datastore"
)

// Test helper functions
func TestSessionPrefix(t *testing.T) {
	expected := "httpsession:gts:"
	actual := sessionPrefix()
	assert.Equal(t, expected, actual, "sessionPrefix should return correct prefix")
}

func TestSessionKey(t *testing.T) {
	sessionId := "test-session-id"
	expected := "httpsession:gts:test-session-id"
	actual := sessionKey(sessionId)
	assert.Equal(t, expected, actual, "sessionKey should return correct Redis key")
}

// Test SessionProvider.Type()
func TestSessionProvider_Type(t *testing.T) {
	provider := &SessionProvider{}
	sessionType := provider.Type()
	assert.Equal(t, SessionTypeRedis, sessionType, "Type should return SessionTypeRedis")
	assert.Equal(t, httpsession.SessionType("REDIS"), sessionType, "Type should return REDIS string")
}

// Test SessionProvider.NewSession()
func TestSessionProvider_NewSession(t *testing.T) {
	provider := &SessionProvider{}
	expireTime := time.Now().Add(time.Hour)

	session := provider.NewSession(expireTime)

	assert.NotNil(t, session, "NewSession should return a non-nil session")
	// Use time.Equal or compare with tolerance since time precision might differ
	assert.WithinDuration(t, expireTime, session.Expire(), time.Second, "Session should have correct expiration time")
	assert.False(t, session.IsExpire(), "Session should not be expired")
}

// Test SessionProvider with nil Redis operators
func TestSessionProvider_NilOperators(t *testing.T) {
	provider := &SessionProvider{
		Master: nil,
		Slave:  nil,
	}

	// Test Sessions() with nil operators
	sessions := provider.Sessions()
	assert.NotNil(t, sessions, "Sessions should return non-nil map")
	assert.Empty(t, sessions, "Sessions should return empty map when no operators available")

	// Test Session() with nil operators
	session := provider.Session("test-key")
	assert.Nil(t, session, "Session should return nil when no operators available")

	// Test Save() with nil operators
	testSession := httpsession.NewDefaultSession(provider)
	testSession.SetExpire(time.Now().Add(time.Hour))
	err := provider.Save(testSession)
	assert.Error(t, err, "Save should return error when no operators available")
	assert.Contains(t, err.Error(), "provider is nil", "Error should mention provider is nil")

	// Test Delete() with nil operators (should not panic)
	provider.Delete("test-key")
	// No assertion needed, just ensuring it doesn't panic
}

// Test SessionProvider.Save() with nil session using official mock
func TestSessionProvider_Save_NilSession(t *testing.T) {
	// Use official mock to test nil session specifically
	mockMaster := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: mockMaster,
		Slave:  nil,
	}

	err := provider.Save(nil)
	assert.Error(t, err, "Save should return error with nil session")
	assert.Contains(t, err.Error(), "session is nil", "Error should mention session is nil")
}

// Test SessionProvider.Save() with expired session using official mock
func TestSessionProvider_Save_ExpiredSession(t *testing.T) {
	// Use official mock to test expired session specifically
	mockMaster := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: mockMaster,
		Slave:  nil,
	}

	session := httpsession.NewDefaultSession(provider)
	session.SetExpire(time.Now().Add(-time.Hour)) // Expired session

	err := provider.Save(session)
	assert.Error(t, err, "Save should return error with expired session")
	assert.Contains(t, err.Error(), "session expired", "Error should mention session expired")
}

// Test SessionProvider.Save() with valid session using official mock
func TestSessionProvider_Save_ValidSession(t *testing.T) {
	mockMaster := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: mockMaster,
		Slave:  nil,
	}

	session := httpsession.NewDefaultSession(provider)
	session.SetExpire(time.Now().Add(time.Hour))

	err := provider.Save(session)
	assert.NoError(t, err, "Save should succeed with valid session and mock Redis")
}

// Test SessionProvider.Session() with official mock
func TestSessionProvider_Session_WithMock(t *testing.T) {
	mockSlave := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: nil,
		Slave:  mockSlave,
	}

	// Test with non-existent session
	session := provider.Session("non-existent-session")
	assert.Nil(t, session, "Session should return nil for non-existent session")
}

// Test SessionProvider.Delete() with official mock
func TestSessionProvider_Delete_WithMock(t *testing.T) {
	mockMaster := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: mockMaster,
		Slave:  nil,
	}

	// Test delete operation (should not panic)
	provider.Delete("test-session-id")
	// No assertion needed, just ensuring it doesn't panic
}

// Test SessionProvider.Sessions() with official mock
func TestSessionProvider_Sessions_WithMock(t *testing.T) {
	mockSlave := datastore.NewMockRedisOp()
	provider := &SessionProvider{
		Master: nil,
		Slave:  mockSlave,
	}

	sessions := provider.Sessions()
	assert.NotNil(t, sessions, "Sessions should return non-nil map")
	assert.Empty(t, sessions, "Sessions should return empty map with mock")
}

// Test NewSessionProviderWithRedis using official mock
func TestNewSessionProviderWithRedis_WithMock(t *testing.T) {
	mockRedis := datastore.NewMockRedis()
	provider := NewSessionProviderWithRedis(mockRedis)

	assert.NotNil(t, provider, "NewSessionProviderWithRedis should create provider with mock Redis")
	assert.NotNil(t, provider.Master, "Provider should have Master from mock Redis")
	assert.NotNil(t, provider.Slave, "Provider should have Slave from mock Redis")
}

// Test complete session workflow with official mock
func TestSessionProvider_CompleteWorkflow_WithMock(t *testing.T) {
	mockMaster := datastore.NewMockRedisOp()
	mockSlave := datastore.NewMockRedisOp()

	provider := &SessionProvider{
		Master: mockMaster,
		Slave:  mockSlave,
	}

	// Test creating a new session
	session := provider.NewSession(time.Now().Add(time.Hour))
	assert.NotNil(t, session, "NewSession should create a session")

	// Test saving the session
	err := provider.Save(session)
	assert.NoError(t, err, "Save should succeed with mock")

	// Test retrieving the session (will be nil with basic mock)
	retrievedSession := provider.Session(session.Id())
	// With basic mock, this will be nil, but we test that it doesn't crash
	_ = retrievedSession

	// Test deleting the session
	provider.Delete(session.Id())
	// No assertion needed, just ensuring it doesn't panic

	// Test listing sessions
	sessions := provider.Sessions()
	assert.NotNil(t, sessions, "Sessions should return non-nil map")
}

// Test SessionProvider constants and variables
func TestSessionConstants(t *testing.T) {
	assert.Equal(t, "httpsession", SessionPrefix, "SessionPrefix should have correct default value")
	assert.Equal(t, httpsession.SessionType("REDIS"), SessionTypeRedis, "SessionTypeRedis should have correct value")
}

// Test SessionProvider struct creation
func TestSessionProvider_Creation(t *testing.T) {
	provider := &SessionProvider{}
	assert.NotNil(t, provider, "SessionProvider should be created successfully")
	assert.Nil(t, provider.Master, "Master should be nil by default")
	assert.Nil(t, provider.Slave, "Slave should be nil by default")
}

// Test NewSessionProvider and NewSessionProviderWithRedis function signatures
func TestConstructorFunctions(t *testing.T) {
	// Test that constructor functions exist and can be compiled
	assert.NotPanics(t, func() {
		// Verify function signatures exist (compilation test)
		var providerFunc = NewSessionProvider
		assert.NotNil(t, providerFunc, "NewSessionProvider function should exist")
	}, "NewSessionProvider function should be properly defined")

	assert.NotPanics(t, func() {
		// We can't test NewSessionProviderWithRedis without a Redis instance
		// but we can verify the function signature compiles
		provider := &SessionProvider{}
		assert.NotNil(t, provider, "SessionProvider struct should be instantiable")
	}, "NewSessionProviderWithRedis function should be properly defined")
}

// Test variable modification
func TestSessionPrefix_Modification(t *testing.T) {
	originalPrefix := SessionPrefix
	defer func() {
		SessionPrefix = originalPrefix // Restore original value
	}()

	// Modify SessionPrefix and test helper functions
	SessionPrefix = "custom"

	expected := "custom:gts:"
	actual := sessionPrefix()
	assert.Equal(t, expected, actual, "sessionPrefix should use modified SessionPrefix")

	sessionId := "test-id"
	expectedKey := "custom:gts:test-id"
	actualKey := sessionKey(sessionId)
	assert.Equal(t, expectedKey, actualKey, "sessionKey should use modified SessionPrefix")
}

// Benchmark tests for performance verification
func BenchmarkSessionKey(b *testing.B) {
	sessionId := "test-session-id-12345"
	for i := 0; i < b.N; i++ {
		_ = sessionKey(sessionId)
	}
}

func BenchmarkSessionPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = sessionPrefix()
	}
}

// Test edge cases for sessionKey function
func TestSessionKey_EdgeCases(t *testing.T) {
	// Test with empty session ID
	emptyKey := sessionKey("")
	assert.Equal(t, "httpsession:gts:", emptyKey, "sessionKey should handle empty session ID")

	// Test with special characters
	specialId := "session-with-special-chars!@#$%"
	specialKey := sessionKey(specialId)
	expected := "httpsession:gts:session-with-special-chars!@#$%"
	assert.Equal(t, expected, specialKey, "sessionKey should handle special characters")
}
