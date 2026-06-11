package sys

import (
	"strings"
	"testing"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestRenderProcessTableIncludesSignatureColumns(t *testing.T) {
	output := renderProcessTable(&implantpb.PsResponse{
		Processes: []*implantpb.Process{
			{
				Name:            "agent.exe",
				Pid:             4242,
				Ppid:            888,
				Arch:            "x64",
				Owner:           `ACME\\operator`,
				Path:            `C:\Tools\agent.exe`,
				Args:            "--stage 1",
				Signed:          true,
				SignatureStatus: "trusted",
				Signer:          "Acme Code Signing",
			},
		},
	})

	for _, want := range []string{"Signed", "Status", "Signer", "trusted", "Acme Code Signing", "yes"} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderProcessTable output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderSysInfoIncludesSignatureDetails(t *testing.T) {
	output := renderSysInfo(&implantpb.SysInfo{
		Filepath: "/opt/agent",
		Workdir:  "/opt",
		Os: &implantpb.Os{
			Name:       "linux",
			Arch:       "x64",
			Hostname:   "ops-host",
			Username:   "operator",
			Locale:     "Asia/Shanghai",
			ClrVersion: []string{"v4.0.30319"},
		},
		Process: &implantpb.Process{
			Name:            "agent",
			Pid:             4242,
			Ppid:            888,
			Arch:            "x64",
			Owner:           "operator",
			Path:            "/opt/agent",
			Args:            "--stage 1",
			Signed:          true,
			SignatureStatus: "trusted",
			Signer:          "Acme Code Signing",
			Issuer:          "Acme Root CA",
		},
	})

	for _, want := range []string{
		"System Info:",
		"clr: v4.0.30319",
		"signature: signed=yes status=trusted signer=Acme Code Signing issuer=Acme Root CA",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderSysInfo output missing %q:\n%s", want, output)
		}
	}
}
