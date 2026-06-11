package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gopkg.in/yaml.v3"
)

func (rpc *Server) AddClient(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	name := req.Args[0]

	// Check if operator with this name already exists
	if existing, _ := db.FindOperatorByName(name); existing != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  fmt.Sprintf("client %q already exists", name),
		}, fmt.Errorf("client %q already exists", name)
	}

	cfg := configs.GetServerConfig()
	clientConf, fingerprint, err := certutils.GenerateClientCert(cfg.IP, name, int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}

	// Determine role: default "operator", or from params if provided
	role := models.RoleOperator
	if r, ok := req.Params["role"]; ok {
		valid := false
		for _, vr := range models.ValidRoles {
			if r == vr {
				valid = true
				break
			}
		}
		if valid {
			role = r
		}
	}

	client := &models.Operator{
		Name:             name,
		Type:             mtls.Client,
		Role:             role,
		Fingerprint:      fingerprint,
		Remote:           getRemoteAddr(ctx),
		CAType:           certs.OperatorCA,
		KeyType:          certs.RSAKey,
		CaCertificatePEM: clientConf.CACertificate,
		CertificatePEM:   clientConf.Certificate,
		PrivateKeyPEM:    clientConf.PrivateKey,
	}
	err = db.CreateOperator(client)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(clientConf)
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: string(data),
	}, nil
}

func (rpc *Server) RemoveClient(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	opCache.InvalidateByName(req.Args[0])
	err := db.RemoveOperator(req.Args[0])
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: "",
	}, nil
}

func (rpc *Server) ListClients(ctx context.Context, req *rootpb.Operator) (*clientpb.Clients, error) {
	operators, err := db.ListClients()
	if err != nil {
		return nil, err
	}
	var clients []*clientpb.Client
	for _, op := range operators {
		client := &clientpb.Client{
			Name: op.Name,
		}
		clients = append(clients, client)
	}
	return &clientpb.Clients{
		Clients: clients,
	}, nil
}

func (rpc *Server) AddListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	name := req.Args[0]

	// Check if operator with this name already exists
	if existing, _ := db.FindOperatorByName(name); existing != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  fmt.Sprintf("listener %q already exists", name),
		}, fmt.Errorf("listener %q already exists", name)
	}

	cfg := configs.GetServerConfig()
	clientConf, fingerprint, err := certutils.GenerateListenerCert(cfg.IP, name, int(cfg.GRPCPort))
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	listener := &models.Operator{
		Name:             name,
		Type:             mtls.Listener,
		Role:             models.RoleListener,
		Fingerprint:      fingerprint,
		Remote:           getRemoteAddr(ctx),
		CAType:           certs.ListenerCA,
		KeyType:          certs.RSAKey,
		CaCertificatePEM: clientConf.CACertificate,
		CertificatePEM:   clientConf.Certificate,
		PrivateKeyPEM:    clientConf.PrivateKey,
	}
	err = db.CreateOperator(listener)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(clientConf)
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: string(data),
	}, nil
}

func (rpc *Server) RemoveListener(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	opCache.InvalidateByName(req.Args[0])
	err := db.RemoveOperator(req.Args[0])
	if err != nil {
		return &rootpb.Response{
			Status: 1,
			Error:  err.Error(),
		}, err
	}
	return &rootpb.Response{
		Status:   0,
		Response: "",
	}, nil
}

func (rpc *Server) ListListeners(ctx context.Context, req *rootpb.Operator) (*clientpb.Listeners, error) {
	dbListeners, err := db.ListListeners()
	if err != nil {
		return nil, err
	}
	listeners := &clientpb.Listeners{}
	for _, listener := range dbListeners {
		listeners.Listeners = append(listeners.Listeners, &clientpb.Listener{
			Id: listener.Name,
			Ip: listener.Remote,
		})
	}

	return listeners, nil
}

// ============================================
// Auth Management RPCs
// ============================================

// SetOperatorRole changes the role of an existing operator.
// Usage: args=[name, role]
func (rpc *Server) SetOperatorRole(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	if len(req.Args) < 2 {
		return &rootpb.Response{Status: 1, Error: "usage: args=[name, role]"}, nil
	}
	name, role := req.Args[0], req.Args[1]

	valid := false
	for _, vr := range models.ValidRoles {
		if vr == role {
			valid = true
			break
		}
	}
	if !valid {
		return &rootpb.Response{
			Status: 1,
			Error:  fmt.Sprintf("invalid role %q, valid roles: %v", role, models.ValidRoles),
		}, nil
	}

	err := db.Session().Model(&models.Operator{}).Where("name = ?", name).Update("role", role).Error
	if err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	opCache.InvalidateByName(name)
	ruleCache.Invalidate()

	return &rootpb.Response{
		Status:   0,
		Response: fmt.Sprintf("role set to %s for %s", role, name),
	}, nil
}

// RevokeOperator sets the revoked flag on an operator, immediately blocking access.
// Usage: args=[name]
func (rpc *Server) RevokeOperator(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	if len(req.Args) < 1 {
		return &rootpb.Response{Status: 1, Error: "usage: args=[name]"}, nil
	}

	err := db.RevokeOperator(req.Args[0])
	if err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	opCache.InvalidateByName(req.Args[0])

	return &rootpb.Response{
		Status:   0,
		Response: fmt.Sprintf("operator %s revoked", req.Args[0]),
	}, nil
}

// ListAuthzRules returns all authorization rules, optionally filtered by role.
// Usage: params["role"] = optional role filter
func (rpc *Server) ListAuthzRules(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	roleFilter := ""
	if req.Params != nil {
		roleFilter = req.Params["role"]
	}

	rules, err := db.ListAuthzRules(roleFilter)
	if err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	type ruleEntry struct {
		ID     string `json:"id"`
		Role   string `json:"role"`
		Method string `json:"method"`
		Allow  bool   `json:"allow"`
	}
	var entries []ruleEntry
	for _, r := range rules {
		entries = append(entries, ruleEntry{
			ID:     r.ID.String(),
			Role:   r.Role,
			Method: r.Method,
			Allow:  r.Allow,
		})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	return &rootpb.Response{Status: 0, Response: string(data)}, nil
}

// AddAuthzRule creates a new authorization rule.
// Usage: args=[role, method_pattern], params["allow"] = "true"|"false" (default "true")
func (rpc *Server) AddAuthzRule(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	if len(req.Args) < 2 {
		return &rootpb.Response{Status: 1, Error: "usage: args=[role, method_pattern]"}, nil
	}

	allow := true
	if req.Params != nil && req.Params["allow"] == "false" {
		allow = false
	}

	rule := &models.AuthzRule{
		Role:   req.Args[0],
		Method: req.Args[1],
		Allow:  allow,
	}

	if err := db.AddAuthzRule(rule); err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	ruleCache.InvalidateRole(rule.Role)

	return &rootpb.Response{
		Status:   0,
		Response: fmt.Sprintf("rule added: role=%s method=%s allow=%v", rule.Role, rule.Method, rule.Allow),
	}, nil
}

// RemoveAuthzRule deletes an authorization rule by ID.
// Usage: args=[rule_id]
func (rpc *Server) RemoveAuthzRule(ctx context.Context, req *rootpb.Operator) (*rootpb.Response, error) {
	if len(req.Args) < 1 {
		return &rootpb.Response{Status: 1, Error: "usage: args=[rule_id]"}, nil
	}

	if err := db.RemoveAuthzRule(req.Args[0]); err != nil {
		return &rootpb.Response{Status: 1, Error: err.Error()}, err
	}

	ruleCache.Invalidate()

	return &rootpb.Response{
		Status:   0,
		Response: fmt.Sprintf("rule %s removed", req.Args[0]),
	}, nil
}
