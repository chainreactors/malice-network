package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
)

const ModuleSchema = "schema"

func SchemaCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := Schema(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, "schema")
	return nil
}

func Schema(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModuleSchema,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{Name: ModuleSchema},
			},
		},
		Expect: "response",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterSchemaFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModuleSchema,
		Schema,
		"",
		nil,
		parseSchemaResponse,
		nil,
	)

	_ = intermediate.RegisterInternalDoneCallback(ModuleSchema, formatSchemaTable)

	_ = con.AddCommandFuncHelper(
		ModuleSchema,
		ModuleSchema,
		ModuleSchema+`(active())`,
		[]string{
			"sess: special session",
		},
		[]string{"task"},
	)
}

type schemaPayload struct {
	Tools []struct {
		Name   string         `json:"name"`
		Schema map[string]any `json:"schema"`
	} `json:"tools"`
}

func parseSchemaResponse(ctx *clientpb.TaskContext) (interface{}, error) {
	if ctx == nil || ctx.Spite == nil {
		return nil, fmt.Errorf("no response")
	}
	resp := ctx.Spite.GetResponse()
	if resp == nil {
		return nil, fmt.Errorf("no response")
	}
	return resp.GetOutput(), nil
}

func formatSchemaTable(ctx *clientpb.TaskContext) (string, error) {
	if ctx == nil || ctx.Spite == nil {
		return "", fmt.Errorf("no response")
	}
	resp := ctx.Spite.GetResponse()
	if resp == nil {
		return "", fmt.Errorf("no response")
	}

	var payload schemaPayload
	if err := json.Unmarshal([]byte(resp.GetOutput()), &payload); err != nil {
		return resp.GetOutput(), nil
	}

	if len(payload.Tools) == 0 {
		return "No tools observed in this session.", nil
	}

	sort.Slice(payload.Tools, func(i, j int) bool {
		return payload.Tools[i].Name < payload.Tools[j].Name
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Observed %d tools:\n\n", len(payload.Tools)))
	sb.WriteString(fmt.Sprintf("  %-4s %-24s %s\n", "#", "Tool", "Required Fields"))
	sb.WriteString(fmt.Sprintf("  %-4s %-24s %s\n", "---", "------------------------", "--------------------"))

	for i, t := range payload.Tools {
		required := extractRequired(t.Schema)
		sb.WriteString(fmt.Sprintf("  %-4d %-24s %s\n", i+1, t.Name, strings.Join(required, ", ")))
	}
	return sb.String(), nil
}

func extractRequired(schema map[string]any) []string {
	arr, ok := schema["required"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
