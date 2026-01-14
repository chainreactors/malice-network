package wizard

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
)

// ProfileOptionsProvider returns a function that fetches profile names from the server
func ProfileOptionsProvider() func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		profiles, err := con.Rpc.GetProfiles(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0, len(profiles.Profiles)+1)
		opts = append(opts, "") // Allow empty (use default profile)
		for _, p := range profiles.Profiles {
			opts = append(opts, p.Name)
		}
		return opts
	}
}

// ListenerOptionsProvider returns a function that fetches listener IDs from the server
func ListenerOptionsProvider() func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		listeners, err := con.Rpc.GetListeners(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0, len(listeners.Listeners))
		for _, l := range listeners.Listeners {
			opts = append(opts, l.Id)
		}
		return opts
	}
}

// PipelineOptionsProvider returns a function that fetches pipeline names from the server
func PipelineOptionsProvider() func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		pipelines, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0, len(pipelines.GetPipelines())+1)
		opts = append(opts, "") // Allow empty
		for _, p := range pipelines.GetPipelines() {
			opts = append(opts, p.Name)
		}
		return opts
	}
}

// ArtifactOptionsProvider returns a function that fetches artifact names from the server
// filterType can be used to filter by artifact type (e.g., "beacon", "pulse", "module")
func ArtifactOptionsProvider(filterType string) func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0, len(artifacts.Artifacts)+1)
		opts = append(opts, "") // Allow empty (no artifact)
		for _, a := range artifacts.Artifacts {
			if filterType == "" || a.Type == filterType {
				opts = append(opts, fmt.Sprintf("%d", a.Id))
			}
		}
		return opts
	}
}

// AddressOptionsProvider returns a function that fetches C2 addresses from pipelines
func AddressOptionsProvider() func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		pipelines, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0)
		opts = append(opts, "") // Allow empty (manual input)
		seen := make(map[string]bool)

		for _, p := range pipelines.GetPipelines() {
			var addr string
			switch body := p.Body.(type) {
			case *clientpb.Pipeline_Tcp:
				tcp := body.Tcp
				if tcp.Host != "" && tcp.Port != 0 {
					addr = fmt.Sprintf("%s:%d", tcp.Host, tcp.Port)
				}
			case *clientpb.Pipeline_Http:
				http := body.Http
				if http.Host != "" && http.Port != 0 {
					schema := "http"
					if p.Tls != nil && p.Tls.Enable {
						schema = "https"
					}
					addr = fmt.Sprintf("%s://%s:%d", schema, http.Host, http.Port)
				}
			}
			if addr != "" && !seen[addr] {
				seen[addr] = true
				opts = append(opts, addr)
			}
		}
		return opts
	}
}

// PulseAddressOptionsProvider returns a function that fetches stage-0 compatible addresses.
// Pulse currently supports only `http://` and `tcp://` targets; HTTPS is intentionally excluded.
func PulseAddressOptionsProvider() func(ctx interface{}) []string {
	return func(ctx interface{}) []string {
		con, ok := ctx.(*core.Console)
		if !ok || con == nil {
			return nil
		}

		pipelines, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
		if err != nil {
			return nil
		}

		opts := make([]string, 0)
		opts = append(opts, "") // Allow empty (manual input)
		seen := make(map[string]bool)

		for _, p := range pipelines.GetPipelines() {
			var addr string
			switch body := p.Body.(type) {
			case *clientpb.Pipeline_Tcp:
				tcp := body.Tcp
				if tcp.Host != "" && tcp.Port != 0 {
					addr = fmt.Sprintf("tcp://%s:%d", tcp.Host, tcp.Port)
				}
			case *clientpb.Pipeline_Http:
				http := body.Http
				if http.Host != "" && http.Port != 0 {
					if p.Tls != nil && p.Tls.Enable {
						continue
					}
					addr = fmt.Sprintf("http://%s:%d", http.Host, http.Port)
				}
			}
			if addr != "" && !seen[addr] {
				seen[addr] = true
				opts = append(opts, addr)
			}
		}
		return opts
	}
}
