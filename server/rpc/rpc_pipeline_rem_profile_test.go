package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// Registering a rem pipeline must not create an automatic <name>_default
// profile: DefaultRemProfile targets the backend tcp pipeline, which the
// register path cannot know, so the auto profile used to nil-panic and kill
// the teamserver process.
func TestRegisterRemPipelineSkipsDefaultProfile(t *testing.T) {
	_ = newRPCTestEnv(t)
	core.Listeners.Add(core.NewListener("rem-register-listener", "127.0.0.1"))

	_, err := (&Server{}).RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Name:       "rem-pipe",
		ListenerId: "rem-register-listener",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-pipe",
				ListenerId: "rem-register-listener",
				Link:       "tcp://127.0.0.1:23456",
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterPipeline for rem pipeline failed: %v", err)
	}
	if _, dbErr := db.FindPipelineByListener("rem-pipe", "rem-register-listener"); dbErr != nil {
		t.Fatalf("rem pipeline missing from DB: %v", dbErr)
	}
	if _, dbErr := db.GetProfileByName("rem-pipe_default"); dbErr == nil {
		t.Fatal("rem pipeline must not get an automatic default profile")
	}
}

// tcp/http pipelines keep the automatic <name>_default profile behavior.
func TestRegisterPipelineDefaultProfileUnchanged(t *testing.T) {
	_ = newRPCTestEnv(t)
	core.Listeners.Add(core.NewListener("default-profile-listener", "127.0.0.1"))

	cases := []struct {
		name     string
		pipeline *clientpb.Pipeline
		address  string
	}{
		{name: "tcp", address: "127.0.0.1:7001", pipeline: &clientpb.Pipeline{
			Name:       "tcp-default-pipe",
			ListenerId: "default-profile-listener",
			Type:       consts.TCPPipeline,
			Body: &clientpb.Pipeline_Tcp{
				Tcp: &clientpb.TCPPipeline{
					Name:       "tcp-default-pipe",
					ListenerId: "default-profile-listener",
					Host:       "0.0.0.0",
					Port:       7001,
				},
			},
		}},
		{name: "http", address: "127.0.0.1:7002", pipeline: &clientpb.Pipeline{
			Name:       "http-default-pipe",
			ListenerId: "default-profile-listener",
			Type:       consts.HTTPPipeline,
			Body: &clientpb.Pipeline_Http{
				Http: &clientpb.HTTPPipeline{
					Name:       "http-default-pipe",
					ListenerId: "default-profile-listener",
					Host:       "0.0.0.0",
					Port:       7002,
				},
			},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := (&Server{}).RegisterPipeline(context.Background(), tc.pipeline); err != nil {
				t.Fatalf("RegisterPipeline failed: %v", err)
			}
			profile, err := db.GetProfileByNameWithConfig(tc.pipeline.Name + "_default")
			if err != nil {
				t.Fatalf("default profile missing: %v", err)
			}
			config, err := implanttypes.LoadProfile(profile.ImplantConfig)
			if err != nil {
				t.Fatalf("default profile implant yaml invalid: %v", err)
			}
			if len(config.Basic.Targets) != 1 || config.Basic.Targets[0].Address != tc.address {
				t.Fatalf("default profile targets = %#v, want %s", config.Basic.Targets, tc.address)
			}
		})
	}
}

// A rem profile created explicitly with params.REMPipeline resolves the rem
// pipeline and the backend tcp pipeline and produces a rem implant config.
func TestNewProfileWithREMPipelineParamBuildsRemProfile(t *testing.T) {
	_ = newRPCTestEnv(t)

	backend := &clientpb.Pipeline{
		Name:       "backend-tcp",
		ListenerId: "rem-profile-listener",
		Ip:         "192.0.2.10",
		Type:       consts.TCPPipeline,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       "backend-tcp",
				ListenerId: "rem-profile-listener",
				Host:       "0.0.0.0",
				Port:       4444,
			},
		},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(backend)); err != nil {
		t.Fatalf("SavePipeline backend failed: %v", err)
	}
	rem := &clientpb.Pipeline{
		Name:       "rem-link",
		ListenerId: "rem-profile-listener",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-link",
				ListenerId: "rem-profile-listener",
				Link:       "tcp://127.0.0.1:23457",
			},
		},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(rem)); err != nil {
		t.Fatalf("SavePipeline rem failed: %v", err)
	}

	err := db.NewProfile(&clientpb.Profile{
		Name:       "rem-explicit",
		PipelineId: "rem-profile-listener:backend-tcp",
		Params:     `{"rem":"rem-link"}`,
	})
	if err != nil {
		t.Fatalf("NewProfile with REMPipeline param failed: %v", err)
	}
	profile, err := db.GetProfileByNameWithConfig("rem-explicit")
	if err != nil {
		t.Fatalf("rem profile missing: %v", err)
	}
	config, err := implanttypes.LoadProfile(profile.ImplantConfig)
	if err != nil {
		t.Fatalf("rem profile implant yaml invalid: %v", err)
	}
	if len(config.Basic.Targets) != 1 {
		t.Fatalf("rem profile targets = %#v, want exactly one", config.Basic.Targets)
	}
	target := config.Basic.Targets[0]
	if target.REM == nil || target.REM.Link != "tcp://127.0.0.1:23457" {
		t.Fatalf("rem profile target REM = %#v, want link tcp://127.0.0.1:23457", target.REM)
	}
	if target.Address != "192.0.2.10:4444" {
		t.Fatalf("rem profile target address = %q, want backend 192.0.2.10:4444", target.Address)
	}
	if config.Implant == nil || !config.Implant.Enable3rd || len(config.Implant.ThirdModules) != 1 || config.Implant.ThirdModules[0] != "rem" {
		t.Fatalf("rem profile implant config = %#v, want enable_3rd with 3rd module rem", config.Implant)
	}
	if config.Pulse == nil || config.Pulse.Target != "192.0.2.10:4444" || config.Pulse.Protocol != consts.TCPPipeline {
		t.Fatalf("rem profile pulse = %#v, want backend tcp target", config.Pulse)
	}
}

// A rem pipeline profiled without params.REMPipeline has no backend tcp
// pipeline; NewProfile must return an error instead of nil-panicking.
func TestNewProfileForRemPipelineWithoutBackendReturnsError(t *testing.T) {
	_ = newRPCTestEnv(t)

	rem := &clientpb.Pipeline{
		Name:       "rem-solo",
		ListenerId: "rem-solo-listener",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-solo",
				ListenerId: "rem-solo-listener",
				Link:       "tcp://127.0.0.1:23458",
			},
		},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(rem)); err != nil {
		t.Fatalf("SavePipeline rem failed: %v", err)
	}

	err := db.NewProfile(&clientpb.Profile{
		Name:       "rem-solo-profile",
		PipelineId: "rem-solo-listener:rem-solo",
	})
	if err == nil || !strings.Contains(err.Error(), "backend tcp pipeline") {
		t.Fatalf("NewProfile error = %v, want a backend-required error", err)
	}
}
