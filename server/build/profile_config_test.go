package build

import (
	"bytes"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestNeedsProfileFiles(t *testing.T) {
	tests := []struct {
		name string
		cfg  *clientpb.BuildConfig
		want bool
	}{
		{
			name: "no profile name",
			cfg:  &clientpb.BuildConfig{},
			want: false,
		},
		{
			name: "all profile data already present",
			cfg: &clientpb.BuildConfig{
				ProfileName:   "tcp_default",
				MaleficConfig: []byte("implant"),
				PreludeConfig: []byte("prelude"),
				Resources:     &clientpb.BuildResources{},
			},
			want: false,
		},
		{
			name: "missing implant config",
			cfg: &clientpb.BuildConfig{
				ProfileName: "tcp_default",
			},
			want: true,
		},
		{
			name: "missing prelude config",
			cfg: &clientpb.BuildConfig{
				ProfileName:   "tcp_default",
				MaleficConfig: []byte("implant"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsProfileFiles(tt.cfg); got != tt.want {
				t.Fatalf("needsProfileFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeProfileFilesPreservesInlineOverrides(t *testing.T) {
	inlineImplant := []byte("inline-implant")
	diskImplant := []byte("disk-implant")
	diskPrelude := []byte("disk-prelude")
	diskResources := &clientpb.BuildResources{
		Entries: []*clientpb.ResourceEntry{
			{Filename: "test.txt", Content: []byte("payload")},
		},
	}
	cfg := &clientpb.BuildConfig{
		ProfileName:   "tcp_default",
		MaleficConfig: inlineImplant,
	}

	mergeProfileFiles(cfg, diskImplant, diskPrelude, diskResources)

	if !bytes.Equal(cfg.MaleficConfig, inlineImplant) {
		t.Fatalf("MaleficConfig = %q, want inline config to be preserved", cfg.MaleficConfig)
	}
	if !bytes.Equal(cfg.PreludeConfig, diskPrelude) {
		t.Fatalf("PreludeConfig = %q, want %q", cfg.PreludeConfig, diskPrelude)
	}
	if cfg.Resources != diskResources {
		t.Fatalf("Resources pointer changed unexpectedly: got %#v, want %#v", cfg.Resources, diskResources)
	}
}
