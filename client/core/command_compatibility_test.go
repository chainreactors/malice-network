package core

import (
	"testing"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/spf13/cobra"
)

func TestCheckCommandCompatibility(t *testing.T) {
	sess := &client.Session{Session: &clientpb.Session{
		Type:    "malefic",
		Os:      &implantpb.Os{Name: "linux", Arch: "amd64"},
		Modules: []string{"execute", "upload"},
	}}

	tests := []struct {
		name        string
		annotations map[string]string
		wantError   bool
	}{
		{name: "matching os list", annotations: map[string]string{"os": "windows, linux"}},
		{name: "os alias", annotations: map[string]string{"os": "linux,darwin"}},
		{name: "arch alias", annotations: map[string]string{"arch": "x64"}},
		{name: "matching implant", annotations: map[string]string{"implant": "malefic"}},
		{name: "all dependencies", annotations: map[string]string{"depend": "execute, upload"}},
		{name: "os substring is not a match", annotations: map[string]string{"os": "linuxkit"}, wantError: true},
		{name: "wrong arch", annotations: map[string]string{"arch": "arm64"}, wantError: true},
		{name: "wrong implant", annotations: map[string]string{"implant": "bind"}, wantError: true},
		{name: "missing dependency", annotations: map[string]string{"depend": "execute, download"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test", Annotations: tt.annotations}
			err := CheckCommandCompatibility(cmd, sess)
			if tt.wantError && err == nil {
				t.Fatal("expected compatibility error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected compatibility error: %v", err)
			}
		})
	}
}

func TestCheckCommandCompatibilityIncludesParentConstraints(t *testing.T) {
	sess := &client.Session{Session: &clientpb.Session{
		Os: &implantpb.Os{Name: "linux", Arch: "x64"},
	}}
	parent := &cobra.Command{Use: "parent", Annotations: map[string]string{"os": "windows"}}
	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)

	if err := CheckCommandCompatibility(child, sess); err == nil {
		t.Fatal("expected inherited compatibility error")
	}
}
