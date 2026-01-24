package core

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/services/localrpc"
	"github.com/chainreactors/malice-network/client/plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// RPC server address for testing
	testRPCAddr = "127.0.0.1:15004"
)

// setupRPCClient creates a gRPC client connection to the test RPC server
func setupRPCClient(t *testing.T) (localrpc.CommandServiceClient, *grpc.ClientConn) {
	t.Helper()

	// These are integration tests; skip when no local RPC server is running.
	if c, err := net.DialTimeout("tcp", testRPCAddr, 250*time.Millisecond); err != nil {
		t.Skipf("Skipping: local RPC server not reachable at %s: %v", testRPCAddr, err)
	} else {
		_ = c.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		testRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Skipf("Skipping: failed to connect to RPC server at %s: %v", testRPCAddr, err)
	}

	client := localrpc.NewCommandServiceClient(conn)
	return client, conn
}

// TestGetSchemas_ExecuteGroup tests getting schemas for execute group
func TestGetSchemas_ExecuteGroup(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "execute",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetSchemas returned error: %s", resp.Error)
	}

	if resp.SchemasJson == "" {
		t.Fatal("GetSchemas returned empty schemas")
	}

	// Verify JSON is valid
	var schemas map[string]map[string]*plugin.CommandSchema
	if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
		t.Fatalf("Failed to unmarshal schemas JSON: %v", err)
	}

	// Verify the execute group exists in the result
	executeSchemas, ok := schemas["execute"]
	if !ok {
		t.Fatal("Execute group not found in schemas")
	}

	if len(executeSchemas) == 0 {
		t.Fatal("No commands found for execute group")
	}

	t.Logf("Execute group: %d commands", len(executeSchemas))

	// Verify each command has proper schema structure
	for cmdName, cmdSchema := range executeSchemas {
		if cmdSchema.Type != "object" {
			t.Errorf("Command %s has invalid type: %s", cmdName, cmdSchema.Type)
		}

		if cmdSchema.Properties == nil {
			t.Errorf("Command %s has nil properties", cmdName)
		}

		// Verify properties have descriptions (from flag Usage)
		for propName, propSchema := range cmdSchema.Properties {
			if propSchema.Type == "" {
				t.Errorf("Property %s.%s has no type", cmdName, propName)
			}
			// Description is optional but should be present for most flags
			if propSchema.Description != "" {
				t.Logf("  %s.%s: %s", cmdName, propName, propSchema.Description)
			}
		}

		t.Logf("  - %s: %d properties", cmdName, len(cmdSchema.Properties))
	}
}

// TestGetSchemas_SysGroup tests getting schemas for sys group
func TestGetSchemas_SysGroup(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "sys",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetSchemas returned error: %s", resp.Error)
	}

	// Verify JSON is valid
	var schemas map[string]map[string]*plugin.CommandSchema
	if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
		t.Fatalf("Failed to unmarshal schemas JSON: %v", err)
	}

	sysSchemas, ok := schemas["sys"]
	if !ok {
		t.Fatal("Sys group not found in schemas")
	}

	if len(sysSchemas) == 0 {
		t.Fatal("No commands found for sys group")
	}

	t.Logf("Sys group: %d commands", len(sysSchemas))
	for cmdName := range sysSchemas {
		t.Logf("  - %s", cmdName)
	}
}

// TestGetSchemas_FileGroup tests getting schemas for file group
func TestGetSchemas_FileGroup(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "file",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetSchemas returned error: %s", resp.Error)
	}

	// Verify JSON is valid
	var schemas map[string]map[string]*plugin.CommandSchema
	if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
		t.Fatalf("Failed to unmarshal schemas JSON: %v", err)
	}

	fileSchemas, ok := schemas["file"]
	if !ok {
		t.Fatal("File group not found in schemas")
	}

	if len(fileSchemas) == 0 {
		t.Fatal("No commands found for file group")
	}

	t.Logf("File group: %d commands", len(fileSchemas))
}

// TestGetSchemas_InvalidGroup tests error handling for invalid group
func TestGetSchemas_InvalidGroup(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "invalid_group_name",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	// Should return error for invalid group
	if resp.Success {
		t.Fatal("GetSchemas should fail for invalid group")
	}

	if resp.Error == "" {
		t.Fatal("GetSchemas should return error message for invalid group")
	}

	t.Logf("Correctly returned error for invalid group: %s", resp.Error)
}

// TestGetSchemas_EmptyGroup tests error handling for empty group
func TestGetSchemas_EmptyGroup(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	// Should return error for empty group
	if resp.Success {
		t.Fatal("GetSchemas should fail for empty group")
	}

	if resp.Error == "" {
		t.Fatal("GetSchemas should return error message for empty group")
	}

	t.Logf("Correctly returned error for empty group: %s", resp.Error)
}

// TestGetSchemas_SchemaStructure tests the detailed structure of returned schemas
func TestGetSchemas_SchemaStructure(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "execute",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetSchemas returned error: %s", resp.Error)
	}

	// Parse schemas
	var schemas map[string]map[string]*plugin.CommandSchema
	if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
		t.Fatalf("Failed to unmarshal schemas JSON: %v", err)
	}

	executeSchemas := schemas["execute"]
	if len(executeSchemas) == 0 {
		t.Fatal("No commands found in execute group")
	}

	// Pick first command to verify structure
	var firstCmd *plugin.CommandSchema
	var firstCmdName string
	for name, schema := range executeSchemas {
		firstCmd = schema
		firstCmdName = name
		break
	}

	t.Logf("Verifying schema structure for command: %s", firstCmdName)

	// Verify CommandSchema structure
	if firstCmd.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", firstCmd.Type)
	}

	if firstCmd.Title == "" {
		t.Error("Title should not be empty")
	}

	if firstCmd.Properties == nil {
		t.Fatal("Properties should not be nil")
	}

	// Verify PropertySchema structure
	for propName, propSchema := range firstCmd.Properties {
		t.Logf("  Property: %s", propName)
		t.Logf("    Type: %s", propSchema.Type)
		t.Logf("    Description: %s", propSchema.Description)
		t.Logf("    Title: %s", propSchema.Title)

		// Verify required fields
		if propSchema.Type == "" {
			t.Errorf("Property %s has empty type", propName)
		}

		// Verify UI hints are present (from default or annotations)
		if propSchema.AdditionalProperties != nil {
			if widget, ok := propSchema.AdditionalProperties["ui:widget"]; ok {
				t.Logf("    UI Widget: %v", widget)
			}
		}
	}

	// Verify metadata
	if firstCmd.XMetadata != nil {
		t.Logf("Metadata:")
		t.Logf("  Name: %s", firstCmd.XMetadata.Name)
		t.Logf("  Plugin: %s", firstCmd.XMetadata.PluginName)
		t.Logf("  TTP: %s", firstCmd.XMetadata.TTP)
		t.Logf("  Opsec: %d", firstCmd.XMetadata.Opsec)
	}
}

// TestGetSchemas_AllGroups tests getting schemas for all available groups
func TestGetSchemas_AllGroups(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	// Test all available groups
	testGroups := []string{"implant", "execute", "sys", "file", "pivot"}

	for _, group := range testGroups {
		t.Run("Group_"+group, func(t *testing.T) {
			req := &localrpc.GetSchemasRequest{
				Group: group,
			}

			resp, err := client.GetSchemas(context.Background(), req)
			if err != nil {
				t.Fatalf("GetSchemas failed for group %s: %v", group, err)
			}

			// Some groups might not have commands, that's ok
			if !resp.Success {
				t.Logf("Group %s: %s", group, resp.Error)
				return
			}

			// Verify JSON is valid
			var schemas map[string]map[string]*plugin.CommandSchema
			if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
				t.Fatalf("Failed to unmarshal schemas JSON for group %s: %v", group, err)
			}

			groupSchemas, ok := schemas[group]
			if !ok {
				t.Fatalf("Group %s not found in schemas", group)
			}

			t.Logf("Group %s: %d commands", group, len(groupSchemas))
			for cmdName := range groupSchemas {
				t.Logf("  - %s", cmdName)
			}
		})
	}
}

// TestGetGroups tests getting all available groups
func TestGetGroups(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetGroupsRequest{}

	resp, err := client.GetGroups(context.Background(), req)
	if err != nil {
		t.Fatalf("GetGroups failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetGroups returned error: %s", resp.Error)
	}

	if len(resp.Groups) == 0 {
		t.Fatal("No groups returned")
	}

	t.Logf("Total groups: %d", len(resp.Groups))

	// Verify each group has valid information
	for groupID, groupTitle := range resp.Groups {
		if groupID == "" {
			t.Error("Group has empty ID")
		}

		if groupTitle == "" {
			t.Errorf("Group %s has empty title", groupID)
		}

		t.Logf("Group: %s, Title: %s", groupID, groupTitle)
	}

	// Verify expected groups exist
	expectedGroups := []string{"implant", "execute", "sys", "file", "pivot"}
	for _, expectedGroup := range expectedGroups {
		if title, ok := resp.Groups[expectedGroup]; ok {
			t.Logf("Found expected group %s with title: %s", expectedGroup, title)
		} else {
			t.Logf("Expected group %s not found (may be empty)", expectedGroup)
		}
	}
}

// TestGetSchemas_VerifySourceMetadata tests that source metadata is included in schemas
func TestGetSchemas_VerifySourceMetadata(t *testing.T) {
	client, conn := setupRPCClient(t)
	defer conn.Close()

	req := &localrpc.GetSchemasRequest{
		Group: "execute",
	}

	resp, err := client.GetSchemas(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("GetSchemas returned error: %s", resp.Error)
	}

	// Parse schemas
	var schemas map[string]map[string]*plugin.CommandSchema
	if err := json.Unmarshal([]byte(resp.SchemasJson), &schemas); err != nil {
		t.Fatalf("Failed to unmarshal schemas JSON: %v", err)
	}

	executeSchemas := schemas["execute"]
	if len(executeSchemas) == 0 {
		t.Fatal("No commands found in execute group")
	}

	// Verify that at least one command has source metadata
	foundSource := false
	for cmdName, cmdSchema := range executeSchemas {
		if cmdSchema.XMetadata != nil && cmdSchema.XMetadata.Source != "" {
			foundSource = true
			t.Logf("Command %s has source: %s", cmdName, cmdSchema.XMetadata.Source)

			// Verify source is one of the expected values
			validSources := map[string]bool{
				"golang":    true,
				"mal":       true,
				"alias":     true,
				"extension": true,
			}

			if !validSources[cmdSchema.XMetadata.Source] {
				t.Errorf("Command %s has invalid source: %s", cmdName, cmdSchema.XMetadata.Source)
			}
		}
	}

	if !foundSource {
		t.Error("No commands have source metadata")
	}
}
