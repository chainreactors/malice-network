//go:build !bridge_agent_proto
// +build !bridge_agent_proto

package llm

import "context"

// CallProvider is a stub when bridge-agent proto support is not compiled in.
// The real implementation (provider.go) uses typed proto parameters.
func CallProvider(_ context.Context, _ ProviderOpts, _ interface{}) interface{} {
	return nil
}
