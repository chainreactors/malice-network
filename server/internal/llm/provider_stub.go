//go:build !bridge_agent_proto
// +build !bridge_agent_proto

package llm

// CallProvider is a stub when bridge-agent proto support is not compiled in.
// It will be replaced by the real implementation once proto types are synced.
func CallProvider(opts ProviderOpts, reqData []byte) ([]byte, string) {
	return nil, "bridge_agent_proto build tag required"
}
