package db

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// SeedDefaultAuthzRules inserts the default role-method mappings if the table is empty.
// This runs at startup and only seeds when no rules exist.
func SeedDefaultAuthzRules() error {
	var count int64
	Session().Model(&models.AuthzRule{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	logs.Log.Infof("seeding default authorization rules")

	rules := []models.AuthzRule{
		// Admin: full access to everything
		{Role: models.RoleAdmin, Method: "/clientrpc.MaliceRPC/*", Allow: true},
		{Role: models.RoleAdmin, Method: "/clientrpc.RootRPC/*", Allow: true},
		{Role: models.RoleAdmin, Method: "/listenerrpc.ListenerRPC/*", Allow: true},

		// Operator: MaliceRPC access (no RootRPC)
		{Role: models.RoleOperator, Method: "/clientrpc.MaliceRPC/*", Allow: true},

		// Listener: ListenerRPC only
		{Role: models.RoleListener, Method: "/listenerrpc.ListenerRPC/*", Allow: true},
	}

	for i := range rules {
		if err := Session().Create(&rules[i]).Error; err != nil {
			return err
		}
	}

	logs.Log.Infof("seeded %d default authorization rules", len(rules))
	return nil
}

// GetAuthzRulesForRole returns all rules for a given role.
func GetAuthzRulesForRole(role string) ([]*models.AuthzRule, error) {
	var rules []*models.AuthzRule
	err := Session().Where("role = ?", role).Find(&rules).Error
	return rules, err
}

// AddAuthzRule adds a new authorization rule.
func AddAuthzRule(rule *models.AuthzRule) error {
	return Session().Create(rule).Error
}

// RemoveAuthzRule removes an authorization rule by ID.
func RemoveAuthzRule(id string) error {
	return Session().Where("id = ?", id).Delete(&models.AuthzRule{}).Error
}

// ListAuthzRules returns all rules, optionally filtered by role.
func ListAuthzRules(role string) ([]*models.AuthzRule, error) {
	var rules []*models.AuthzRule
	query := Session().Model(&models.AuthzRule{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	return rules, query.Find(&rules).Error
}
