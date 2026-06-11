package models

import (
	"testing"

	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestWebsiteContentURL_NormalizesRootAndContentPath(t *testing.T) {
	tests := []struct {
		name string
		root string
		path string
		tls  bool
		want string
	}{
		{
			name: "root slash plus content slash",
			root: "/",
			path: "/1",
			want: "http://192.168.239.161:8081/1",
		},
		{
			name: "nested root plus content slash",
			root: "/files",
			path: "/payload.bin",
			want: "http://192.168.239.161:8081/files/payload.bin",
		},
		{
			name: "https website",
			root: "/",
			path: "/index.html",
			tls:  true,
			want: "https://192.168.239.161:8081/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := &WebsiteContent{
				Path: tt.path,
				Pipeline: &Pipeline{
					IP:   "192.168.239.161",
					Port: 8081,
					PipelineParams: &implanttypes.PipelineParams{
						WebPath: tt.root,
						Tls: &implanttypes.TlsConfig{
							Enable: tt.tls,
						},
					},
				},
			}

			if got := wc.URL(); got != tt.want {
				t.Fatalf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}
