package store

import (
	"sync"
)

// DeviceStore はデバイストークンを管理します。
type DeviceStore struct {
	mu     sync.Mutex
	tokens map[string]bool // トークンをキーとして、存在確認を容易にするためにbool値を格納
}

// NewDeviceStore は新しいDeviceStoreのインスタンスを作成します。
func NewDeviceStore() *DeviceStore {
	return &DeviceStore{
		tokens: make(map[string]bool),
	}
}

// AddToken は新しいデバイストークンを追加します。
// トークンが新しく追加された場合は true を、既に存在した場合は false を返します。
func (s *DeviceStore) AddToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tokens[token]; exists {
		return false // 既に存在した
	}
	s.tokens[token] = true
	return true // 新しく追加された
}

// GetTokens は登録されているすべてのデバイストークンを返します。
func (s *DeviceStore) GetTokens() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var tokens []string
	for token := range s.tokens {
		tokens = append(tokens, token)
	}
	return tokens
}

// RemoveToken は指定されたデバイストークンを削除します。
func (s *DeviceStore) RemoveToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}
