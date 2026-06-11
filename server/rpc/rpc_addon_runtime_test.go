package rpc

import (
	"context"
	"strings"
	"testing"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func TestAddonHandlersRejectNilRequest(t *testing.T) {
	server := &Server{}

	if _, err := server.ListAddon(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ListAddon(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.LoadAddon(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("LoadAddon(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.ExecuteAddon(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ExecuteAddon(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestApplyAddonsResponseDeduplicatesAndPersistsAddons(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-addon-list", "rpc-addon-list-pipe", true)
	sess.Addons = []*implantpb.Addon{
		{Name: "seatbelt", Type: "bof", Depend: "exec"},
	}

	applyAddonsResponse(sess, &implantpb.Spite{
		Body: &implantpb.Spite_Addons{
			Addons: &implantpb.Addons{Addons: []*implantpb.Addon{
				nil,
				{Name: "seatbelt", Type: "bof", Depend: "exec"},
				{Name: "", Type: "skip", Depend: "exec"},
				{Name: "sharpview", Type: "assembly", Depend: "exec"},
				{Name: "sharpview", Type: "assembly", Depend: "exec"},
			}},
		},
	}, false)

	if len(sess.Addons) != 2 {
		t.Fatalf("runtime addons = %#v, want 2 unique addons", sess.Addons)
	}
	if addon := findAddon(sess.Addons, "seatbelt"); addon == nil || addon.GetType() != "bof" {
		t.Fatalf("runtime seatbelt addon = %#v, want bof addon", addon)
	}
	if addon := findAddon(sess.Addons, "sharpview"); addon == nil || addon.GetType() != "assembly" {
		t.Fatalf("runtime sharpview addon = %#v, want assembly addon", addon)
	}

	stored, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("getSession failed: %v", err)
	}
	if len(stored.GetAddons()) != 2 {
		t.Fatalf("stored addons = %#v, want 2 unique addons", stored.GetAddons())
	}
}

func TestApplyAddonLoadDeduplicatesRepeatedLoadsAndRefreshesMetadata(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-addon-load", "rpc-addon-load-pipe", true)
	sess.Addons = []*implantpb.Addon{
		{Name: "seatbelt", Type: "bof", Depend: "exec"},
	}

	applyAddonLoad(sess, &implantpb.LoadAddon{Name: "seatbelt", Type: "assembly", Depend: "execute_dll"})
	applyAddonLoad(sess, &implantpb.LoadAddon{Name: "sharpview", Type: "assembly", Depend: "exec"})
	applyAddonLoad(sess, &implantpb.LoadAddon{Name: "sharpview", Type: "assembly", Depend: "exec"})

	if len(sess.Addons) != 2 {
		t.Fatalf("runtime addons = %#v, want 2 unique addons", sess.Addons)
	}
	seatbelt := findAddon(sess.Addons, "seatbelt")
	if seatbelt == nil || seatbelt.GetType() != "assembly" || seatbelt.GetDepend() != "execute_dll" {
		t.Fatalf("runtime seatbelt addon = %#v, want refreshed metadata", seatbelt)
	}

	stored, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("getSession failed: %v", err)
	}
	if len(stored.GetAddons()) != 2 {
		t.Fatalf("stored addons = %#v, want 2 unique addons", stored.GetAddons())
	}
	seatbelt = findAddon(stored.GetAddons(), "seatbelt")
	if seatbelt == nil || seatbelt.GetType() != "assembly" || seatbelt.GetDepend() != "execute_dll" {
		t.Fatalf("stored seatbelt addon = %#v, want refreshed metadata", seatbelt)
	}
}

func findAddon(addons []*implantpb.Addon, name string) *implantpb.Addon {
	for _, addon := range addons {
		if addon != nil && addon.GetName() == name {
			return addon
		}
	}
	return nil
}
