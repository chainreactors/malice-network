package rpc

import (
	"strings"
	"sync"

	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// ============================================
// Operator Cache
// ============================================

// operatorCache is a simple in-memory cache for operator lookups.
// Uses sync.RWMutex for concurrent read access. Cache misses trigger DB queries.
// Invalidation is explicit (called on revoke/role change/delete).
type operatorCache struct {
	mu     sync.RWMutex
	byFP   map[string]*models.Operator
	byName map[string]*models.Operator
}

var opCache = &operatorCache{
	byFP:   make(map[string]*models.Operator),
	byName: make(map[string]*models.Operator),
}

// LookupByFingerprint returns the operator for the given cert fingerprint.
// Returns (operator, true) on hit, (nil, false) on miss.
func (c *operatorCache) LookupByFingerprint(fp string) (*models.Operator, bool) {
	c.mu.RLock()
	op, ok := c.byFP[fp]
	c.mu.RUnlock()
	if ok {
		return op, true
	}

	// Cache miss: acquire write lock and double-check before querying DB.
	c.mu.Lock()
	defer c.mu.Unlock()
	if op, ok := c.byFP[fp]; ok {
		return op, true
	}
	op, err := db.FindOperatorByFingerprint(fp)
	if err != nil {
		return nil, false
	}
	c.byFP[fp] = op
	c.byName[op.Name] = op
	return op, true
}

// LookupByName returns the operator for the given name.
func (c *operatorCache) LookupByName(name string) (*models.Operator, bool) {
	c.mu.RLock()
	op, ok := c.byName[name]
	c.mu.RUnlock()
	if ok {
		return op, true
	}

	// Cache miss: acquire write lock and double-check before querying DB.
	c.mu.Lock()
	defer c.mu.Unlock()
	if op, ok := c.byName[name]; ok {
		return op, true
	}
	op, err := db.FindOperatorByName(name)
	if err != nil {
		return nil, false
	}
	c.byFP[op.Fingerprint] = op
	c.byName[op.Name] = op
	return op, true
}

// InvalidateByName removes a specific operator from cache by name.
func (c *operatorCache) InvalidateByName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if op, ok := c.byName[name]; ok {
		delete(c.byFP, op.Fingerprint)
		delete(c.byName, name)
	}
}

// Invalidate clears the entire operator cache.
func (c *operatorCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byFP = make(map[string]*models.Operator)
	c.byName = make(map[string]*models.Operator)
}

// ============================================
// AuthzRule Cache
// ============================================

// authzRuleCache caches authorization rules by role.
type authzRuleCache struct {
	mu    sync.RWMutex
	rules map[string][]*models.AuthzRule // role -> rules
}

var ruleCache = &authzRuleCache{
	rules: make(map[string][]*models.AuthzRule),
}

// GetRules returns all authz rules for a given role.
// Cache miss triggers DB query.
func (c *authzRuleCache) GetRules(role string) []*models.AuthzRule {
	c.mu.RLock()
	rules, ok := c.rules[role]
	c.mu.RUnlock()
	if ok {
		return rules
	}

	// Cache miss: acquire write lock and double-check before querying DB.
	c.mu.Lock()
	defer c.mu.Unlock()
	if rules, ok := c.rules[role]; ok {
		return rules
	}
	rules, err := db.GetAuthzRulesForRole(role)
	if err != nil {
		return nil
	}
	c.rules[role] = rules
	return rules
}

// InvalidateRole removes cached rules for a specific role.
func (c *authzRuleCache) InvalidateRole(role string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rules, role)
}

// Invalidate clears the entire rule cache.
func (c *authzRuleCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules = make(map[string][]*models.AuthzRule)
}

// matchMethod checks if a gRPC method matches a rule pattern.
// Supports:
//   - Exact match: "/clientrpc.MaliceRPC/GetSessions"
//   - Service wildcard: "/clientrpc.MaliceRPC/*" matches "/clientrpc.MaliceRPC/GetSessions"
//   - Package prefix: "/listenerrpc.*" matches "/listenerrpc.ListenerRPC/SpiteStream"
func matchMethod(pattern, method string) bool {
	if pattern == method {
		return true
	}
	// "/clientrpc.MaliceRPC/*" → prefix "/clientrpc.MaliceRPC/"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(method, prefix)
	}
	// "/listenerrpc.*" → prefix "/listenerrpc."
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(method, prefix)
	}
	return false
}
