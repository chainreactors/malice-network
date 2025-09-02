package core

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

// SecureManager 专注于密钥交换请求和响应处理
type SecureManager struct {
	// KeyPair 引用（来自 SessionContext，不需要复制）
	sessionID string

	keyPair *clientpb.KeyPair
	// 轮换相关字段
	keyCounter    uint32 // 消息计数器
	rotationCount uint32 // 轮换消息计数
}

// NewSecureSpiteManager 创建新的安全管理器
func NewSecureSpiteManager(sess *Session) *SecureManager {
	return &SecureManager{
		keyPair:       sess.KeyPair,
		sessionID:     sess.ID,
		keyCounter:    0,
		rotationCount: 100, // 默认100次交互
	}
}

// UpdateKeyPair 更新KeyPair引用
func (s *SecureManager) UpdateKeyPair(keyPair *clientpb.KeyPair) {
	s.keyPair = keyPair
}

// ShouldRotateKey 检查是否需要轮换密钥
func (s *SecureManager) ShouldRotateKey() bool {
	// 检查交互计数
	return s.keyCounter >= s.rotationCount
}

// IncrementCounter 增加消息计数器
func (s *SecureManager) IncrementCounter() {
	s.keyCounter++
}

// ResetCounters 重置计数器（密钥交换完成后调用）
func (s *SecureManager) ResetCounters() {
	s.keyCounter = 0
}
