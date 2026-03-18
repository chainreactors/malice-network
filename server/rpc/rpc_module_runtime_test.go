package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func TestModuleHandlersRejectNilRequest(t *testing.T) {
	server := &Server{}

	if _, err := server.ListModule(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ListModule(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.LoadModule(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("LoadModule(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.RefreshModule(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RefreshModule(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.ExecuteModule(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ExecuteModule(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.ExecuteModule(context.Background(), &implantpb.ExecuteModuleRequest{}); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ExecuteModule(missing spite) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestApplyModulesResponseIgnoresMissingModules(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-module-missing", "rpc-module-pipe", true)
	sess.Modules = []string{"mod-a"}

	applyModulesResponse(sess, nil, false)
	applyModulesResponse(sess, &implantpb.Spite{Name: consts.ModuleListModule}, false)

	if len(sess.Modules) != 1 || sess.Modules[0] != "mod-a" {
		t.Fatalf("modules = %#v, want unchanged modules", sess.Modules)
	}
}

func TestApplyModulesResponseReplacesOrAppendsModules(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-module-apply", "rpc-module-pipe", true)
	sess.Modules = []string{"mod-a"}

	applyModulesResponse(sess, &implantpb.Spite{
		Body: &implantpb.Spite_Modules{
			Modules: &implantpb.Modules{Modules: []string{"mod-b"}},
		},
	}, false)
	if len(sess.Modules) != 1 || sess.Modules[0] != "mod-b" {
		t.Fatalf("replace modules = %#v, want [mod-b]", sess.Modules)
	}

	applyModulesResponse(sess, &implantpb.Spite{
		Body: &implantpb.Spite_Modules{
			Modules: &implantpb.Modules{Modules: []string{"mod-c", "mod-d"}},
		},
	}, true)
	if want := []string{"mod-b", "mod-c", "mod-d"}; strings.Join(sess.Modules, ",") != strings.Join(want, ",") {
		t.Fatalf("append modules = %#v, want %#v", sess.Modules, want)
	}
}
