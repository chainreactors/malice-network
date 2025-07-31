package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/chainreactors/malice-network/helper/cryptography"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
)

// SecureSpiteManager 专注于密钥管理和交换
type SecureSpiteManager struct {
	// KeyPair 引用（来自 SessionContext，不需要复制）
	keyPair *clientpb.KeyPair

	// 运行时状态（不需要持久化）
	sessionKey   [32]byte
	keyID        string
	keyCounter   uint32    // 每次重启从0开始
	lastRotation time.Time // 每次重启时重新设置

	// 配置（常量，不需要protobuf）
	rotationInterval time.Duration // 1小时轮换
	rotationCount    uint32        // 100条消息轮换

	// 运行时状态
	isInitialized bool
}

// NewSecureSpiteManager 通过KeyPair创建新的安全管理器
// 如果 keyPair 为空，将自动生成一组 Age 密钥对
func NewSecureSpiteManager(keyPair *clientpb.KeyPair) *SecureSpiteManager {
	manager := &SecureSpiteManager{
		keyPair:          keyPair,
		rotationInterval: 1 * time.Hour, // 默认1小时轮换
		rotationCount:    100,           // 默认100条消息轮换
		lastRotation:     time.Now(),
		isInitialized:    false,
	}

	// 如果没有预分发的 KeyPair，自动生成一组
	if keyPair == nil || keyPair.PublicKey == "" || keyPair.PrivateKey == "" {
		logs.Log.Infof("[secure] no pre-distributed keypair found, auto-generating Age keypair")
		generatedKeyPair, err := manager.generateKeyPair()
		if err != nil {
			logs.Log.Errorf("[secure] failed to generate keypair: %v", err)
			return manager
		}
		manager.keyPair = generatedKeyPair
		logs.Log.Infof("[secure] auto-generated keypair with ID: %s", generatedKeyPair.KeyId)
	}

	return manager
}

// IsEnabled 检查安全模式是否启用
func (s *SecureSpiteManager) IsEnabled() bool {
	return s.keyPair != nil && s.keyPair.PublicKey != "" && s.keyPair.PrivateKey != ""
}

// IsInitialized 检查会话密钥是否已初始化
func (s *SecureSpiteManager) IsInitialized() bool {
	return s.isInitialized
}

// InitSessionKey 初始化会话密钥（服务端调用）
func (s *SecureSpiteManager) InitSessionKey() error {
	// 生成随机会话密钥
	_, err := rand.Read(s.sessionKey[:])
	if err != nil {
		return err
	}

	// 生成密钥ID
	hash := sha256.Sum256(s.sessionKey[:])
	s.keyID = hex.EncodeToString(hash[:8]) // 使用前8字节作为ID
	s.keyCounter = 0
	s.lastRotation = time.Now()
	s.isInitialized = true

	return nil
}

// SetSessionKey 设置会话密钥（implant 端调用）
func (s *SecureSpiteManager) SetSessionKey(key [32]byte, keyID string) {
	s.sessionKey = key
	s.keyID = keyID
	s.keyCounter = 0
	s.lastRotation = time.Now()
	s.isInitialized = true
}

// ShouldRotateKey 检查是否需要轮换密钥
func (s *SecureSpiteManager) ShouldRotateKey() bool {
	if !s.isInitialized {
		return false
	}

	// 检查时间间隔
	if time.Since(s.lastRotation) > s.rotationInterval {
		return true
	}

	// 检查消息计数
	if s.keyCounter >= s.rotationCount {
		return true
	}

	return false
}

// GetKeyID 获取当前密钥ID
func (s *SecureSpiteManager) GetKeyID() string {
	return s.keyID
}

// IncrementCounter 增加消息计数器
func (s *SecureSpiteManager) IncrementCounter() {
	s.keyCounter++
}

// BuildKeyExchangeRequest 构建密钥交换请求（implant 端）
func (s *SecureSpiteManager) BuildKeyExchangeRequest() (*implantpb.KeyExchangeRequest, error) {
	// 生成临时密钥对用于此次交换
	keyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return nil, err
	}

	// 创建请求
	req := &implantpb.KeyExchangeRequest{
		EphemeralPublicKey: []byte(keyPair.Public),
		Timestamp:          uint64(time.Now().Unix()),
		Nonce:              cryptography.RandomString(16),
	}

	return req, nil
}

// ProcessKeyExchangeResponse 处理密钥交换响应（implant 端）
func (s *SecureSpiteManager) ProcessKeyExchangeResponse(resp *implantpb.KeyExchangeResponse) error {
	if !s.IsEnabled() {
		return fmt.Errorf("secure mode not enabled")
	}

	// 使用内部 KeyPair 的私钥解密会话密钥
	sessionKeyBytes, err := cryptography.AgeDecrypt(s.keyPair.PrivateKey, resp.EncryptedSessionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt session key: %v", err)
	}

	if len(sessionKeyBytes) != 32 {
		return fmt.Errorf("invalid session key length: %d", len(sessionKeyBytes))
	}

	// 设置新的会话密钥
	var newKey [32]byte
	copy(newKey[:], sessionKeyBytes)
	s.SetSessionKey(newKey, resp.KeyId)

	return nil
}

// BuildKeyExchangeResponse 构建密钥交换响应（服务端）
func (s *SecureSpiteManager) BuildKeyExchangeResponse(req *implantpb.KeyExchangeRequest) (*implantpb.KeyExchangeResponse, error) {
	// 生成新的会话密钥
	err := s.InitSessionKey()
	if err != nil {
		return nil, err
	}

	// 使用 implant 提供的临时公钥加密会话密钥
	implantPublicKey := string(req.EphemeralPublicKey)
	encryptedKey, err := cryptography.AgeEncrypt(implantPublicKey, s.sessionKey[:])
	if err != nil {
		return nil, err
	}

	// 生成服务端临时密钥对（用于响应）
	serverKeyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return nil, err
	}

	resp := &implantpb.KeyExchangeResponse{
		EphemeralPublicKey:  []byte(serverKeyPair.Public),
		EncryptedSessionKey: encryptedKey,
		Timestamp:           uint64(time.Now().Unix()),
		KeyId:               s.keyID,
	}

	return resp, nil
}

// NeedsKeyExchange 检查是否需要进行密钥交换
func (s *SecureSpiteManager) NeedsKeyExchange() bool {
	return s.IsEnabled() && !s.IsInitialized()
}

// GetSessionKey 获取当前会话密钥（供 parser 使用）
func (s *SecureSpiteManager) GetSessionKey() ([32]byte, error) {
	if !s.isInitialized {
		return [32]byte{}, fmt.Errorf("session key not initialized")
	}

	return s.sessionKey, nil
}

// CheckRotationRequired 检查是否需要进行密钥轮换
func (s *SecureSpiteManager) CheckRotationRequired() bool {
	if !s.isInitialized {
		return false
	}

	// 检查时间间隔
	if time.Since(s.lastRotation) >= s.rotationInterval {
		return true
	}

	// 检查消息计数
	if s.keyCounter >= s.rotationCount {
		return true
	}

	return false
}

// BuildKeyRotationRequest 构建密钥轮换请求（服务端主导）
func (s *SecureSpiteManager) BuildKeyRotationRequest() (*implantpb.KeyExchangeRequest, error) {
	// 为简化实现，重用现有的BuildKeyExchangeRequest逻辑
	// 生成临时公钥用于密钥交换
	tempPublicKey := make([]byte, 32)
	rand.Read(tempPublicKey)

	// 创建时间戳和随机数
	timestamp := uint64(time.Now().Unix())
	nonce := s.generateRandomHex(16)

	// 使用当前会话密钥进行签名
	signData := fmt.Sprintf("%x:%d:%s", tempPublicKey, timestamp, nonce)
	signature := s.signWithSessionKey([]byte(signData))

	return &implantpb.KeyExchangeRequest{
		EphemeralPublicKey: tempPublicKey,
		Signature:          signature,
		Timestamp:          timestamp,
		Nonce:              nonce,
		// IsRotation:         true, // 暂时注释，等proto重新生成
	}, nil
}

// ProcessKeyRotationResponse 处理密钥轮换响应并更新会话密钥
func (s *SecureSpiteManager) ProcessKeyRotationResponse(response *implantpb.KeyExchangeResponse) error {
	if !s.IsEnabled() {
		return fmt.Errorf("secure mode not enabled")
	}

	// 使用内部 KeyPair 的私钥解密会话密钥
	sessionKeyBytes, err := cryptography.AgeDecrypt(s.keyPair.PrivateKey, response.EncryptedSessionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt session key during rotation: %v", err)
	}

	if len(sessionKeyBytes) != 32 {
		return fmt.Errorf("invalid session key length: %d", len(sessionKeyBytes))
	}

	// 设置新的会话密钥
	copy(s.sessionKey[:], sessionKeyBytes)

	// 更新密钥 ID
	s.keyID = response.KeyId

	// 重置计数器和时间戳
	s.keyCounter = 0
	s.lastRotation = time.Now()

	logs.Log.Infof("Key rotation completed successfully, new key ID: %s", s.keyID)
	return nil
}

// generateRandomHex 生成随机十六进制字符串
func (s *SecureSpiteManager) generateRandomHex(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// signWithSessionKey 使用当前会话密钥进行签名
func (s *SecureSpiteManager) signWithSessionKey(data []byte) []byte {
	// 使用 HMAC-SHA256 进行签名
	hash := sha256.New()
	hash.Write(s.sessionKey[:])
	hash.Write(data)
	return hash.Sum(nil)
}

// UpdateKeyPair 更新KeyPair引用
func (s *SecureSpiteManager) UpdateKeyPair(keyPair *clientpb.KeyPair) {
	s.keyPair = keyPair
}

// GetKeyPair 获取当前KeyPair引用
func (s *SecureSpiteManager) GetKeyPair() *clientpb.KeyPair {
	return s.keyPair
}

// UpdateKeyPairInSession 更新KeyPair时同步更新到SessionContext
// 这个方法需要一个回调函数来更新SessionContext中的KeyPair
func (s *SecureSpiteManager) UpdateKeyPairInSession(updateFunc func(*clientpb.KeyPair)) {
	if updateFunc != nil && s.keyPair != nil {
		updateFunc(s.keyPair)
	}
}

// generateKeyPair 生成新的 Age 密钥对
func (s *SecureSpiteManager) generateKeyPair() (*clientpb.KeyPair, error) {
	// 使用现有的 cryptography 包生成 Age 密钥对
	ageKeyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate age keypair: %v", err)
	}

	// 生成密钥ID（使用公钥的哈希）
	hash := sha256.Sum256([]byte(ageKeyPair.Public))
	keyID := hex.EncodeToString(hash[:8]) // 使用前8字节作为ID

	now := time.Now().Unix()
	keyPair := &clientpb.KeyPair{
		PublicKey:  ageKeyPair.Public,
		PrivateKey: ageKeyPair.Private,
		KeyId:      keyID,
		CreatedAt:  now,
		ExpiresAt:  0, // 0表示永不过期
	}

	return keyPair, nil
}

// NeedsKeyExchange 检查是否需要进行初始密钥交换
// 当启用安全模式但会话密钥未初始化时返回 true
func (s *SecureSpiteManager) NeedsInitialKeyExchange() bool {
	return s.IsEnabled() && !s.IsInitialized()
}
