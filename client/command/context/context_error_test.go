package context

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	clientcore "github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
)

func TestAddDownloadRequiresSession(t *testing.T) {
	con := &clientcore.Console{Log: iomclient.Log}

	if _, err := AddDownload(con, nil, &clientpb.Task{}, &output.FileDescriptor{}); err == nil {
		t.Fatal("expected AddDownload to fail when session is nil")
	}
}

func TestAddDownloadRequiresTask(t *testing.T) {
	con := &clientcore.Console{Log: iomclient.Log}
	sess := &iomclient.Session{
		Session: &clientpb.Session{SessionId: "sess-1"},
	}

	if _, err := AddDownload(con, sess, nil, &output.FileDescriptor{}); err == nil {
		t.Fatal("expected AddDownload to fail when task is nil")
	}
}
