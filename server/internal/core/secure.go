package core

import (
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

// SecureManager 专注于密钥交换请求和响应处理
type SecureManager struct {
	// KeyPair 引用（来自 SessionContext，不需要复制）
	sessionID string

	keyPair *clientpb.KeyPair
	// 轮换相关字段
	keyCounter   uint32    // 消息计数器
	lastRotation time.Time // 上次轮换时间

	// 轮换配置
	rotationInterval time.Duration // 轮换时间间隔
	rotationCount    uint32        // 轮换消息计数
}

// NewSecureSpiteManager 创建新的安全管理器
func NewSecureSpiteManager(sess *Session) *SecureManager {
	return &SecureManager{
		keyPair:          sess.KeyPair,
		sessionID:        sess.ID,
		keyCounter:       0,
		lastRotation:     time.Now(),
		rotationInterval: time.Duration(sess.Interval*100) * time.Millisecond,
		rotationCount:    100, // 默认100条消息
	}
}

// UpdateKeyPair 更新KeyPair引用
func (s *SecureManager) UpdateKeyPair(keyPair *clientpb.KeyPair) {
	s.keyPair = keyPair
}

// ShouldRotateKey 检查是否需要轮换密钥
func (s *SecureManager) ShouldRotateKey() bool {
	// 检查消息计数
	if s.keyCounter >= s.rotationCount {
		return true
	}

	// 检查时间间隔
	//if time.Since(s.lastRotation) >= s.rotationInterval {
	//	return true
	//}

	return false
}

// IncrementCounter 增加消息计数器
func (s *SecureManager) IncrementCounter() {
	s.keyCounter++
}

// ResetCounters 重置计数器和时间（密钥交换完成后调用）
func (s *SecureManager) ResetCounters() {
	s.keyCounter = 0
	s.lastRotation = time.Now()
}

// SetRotationInterval 设置轮换间隔（100倍session interval）
func (s *SecureManager) SetRotationInterval(sessionInterval time.Duration) {
	s.rotationInterval = sessionInterval * 100
	logs.Log.Debugf("Set key rotation interval to %v (100x session interval)", s.rotationInterval)
}
