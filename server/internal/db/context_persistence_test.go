package db

import (
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/gofrs/uuid"
)

func TestSaveContextDerivesAssociationsFromTaskOnly(t *testing.T) {
	initTestDB(t)

	if err := Session().Create(&models.Session{SessionID: "ctx-task-only"}).Error; err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	if err := Session().Create(&models.Task{ID: "ctx-task-only-7", SessionID: "ctx-task-only", Seq: 7, Type: "credential"}).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	model, err := SaveContext(&clientpb.Context{
		Task:  &clientpb.Task{SessionId: "ctx-task-only", TaskId: 7},
		Type:  consts.ContextCredential,
		Nonce: "nonce-7",
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "host.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}

	if model.SessionID != "ctx-task-only" {
		t.Fatalf("session id = %q, want ctx-task-only", model.SessionID)
	}
	if model.TaskID != "ctx-task-only-7" {
		t.Fatalf("task id = %q, want ctx-task-only-7", model.TaskID)
	}
	if model.Nonce != "nonce-7" {
		t.Fatalf("nonce = %q, want nonce-7", model.Nonce)
	}

	pb := model.ToProtobuf()
	if pb.Nonce != "nonce-7" {
		t.Fatalf("protobuf nonce = %q, want nonce-7", pb.Nonce)
	}
}

func TestFindContextSupportsPrefixAndAmbiguity(t *testing.T) {
	initTestDB(t)

	value := output.MarshalContext(&output.CredentialContext{
		CredentialType: output.UserPassCredential,
		Target:         "host.local",
		Params: map[string]string{
			"username": "alice",
			"password": "secret",
		},
	})

	firstID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	secondID := uuid.FromStringOrNil("11111111-2222-2222-2222-222222222222")
	if err := Session().Create(&models.Context{ID: firstID, Type: consts.ContextCredential, Value: value}).Error; err != nil {
		t.Fatalf("Create first context failed: %v", err)
	}
	if err := Session().Create(&models.Context{ID: secondID, Type: consts.ContextCredential, Value: value}).Error; err != nil {
		t.Fatalf("Create second context failed: %v", err)
	}

	found, err := FindContext("11111111-1111")
	if err != nil {
		t.Fatalf("FindContext unique prefix failed: %v", err)
	}
	if found.ID != firstID {
		t.Fatalf("found id = %s, want %s", found.ID, firstID)
	}

	if _, err := FindContext("11111111"); err == nil || !strings.Contains(err.Error(), "ambiguous context prefix") {
		t.Fatalf("FindContext ambiguous prefix error = %v, want ambiguous prefix error", err)
	}
}
