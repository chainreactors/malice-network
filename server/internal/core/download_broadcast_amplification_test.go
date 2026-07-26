package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestDownloadChunkBroadcastDoesNotCarryFullFileContent(t *testing.T) {
	origUpdateCur := taskDBUpdateCur
	origUpdateFinish := taskDBUpdateFinish
	taskDBUpdateCur = func(string, int) error { return nil }
	taskDBUpdateFinish = func(string) error { return nil }
	t.Cleanup(func() {
		taskDBUpdateCur = origUpdateCur
		taskDBUpdateFinish = origUpdateFinish
	})

	origBroker := EventBroker
	broker := newTestBroker()
	EventBroker = broker
	broker.Start()
	t.Cleanup(func() {
		broker.Stop()
		EventBroker = origBroker
	})

	deadline := time.After(2 * time.Second)
	for !broker.alive.Load() {
		select {
		case <-deadline:
			t.Fatal("broker did not become alive")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	sub, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe error = %v", err)
	}
	t.Cleanup(func() { broker.Unsubscribe(sub) })

	const (
		chunkSize = 1 << 20 // 1 MiB per chunk (matches the packet_length order)
		numChunks = 8       // an 8 MiB "download"
	)
	fileSize := chunkSize * numChunks
	checksum := "download-checksum"

	sess := newTestSession("download-amp")
	task := sess.NewTask("download", numChunks)

	assertEvent := func(expectedOp string, expectedCur int32) {
		t.Helper()
		select {
		case event := <-sub:
			if event.Op != expectedOp {
				t.Fatalf("event op = %q, want %q", event.Op, expectedOp)
			}
			download := event.Spite.GetDownloadResponse()
			if download == nil {
				t.Fatal("event download response is nil")
			}
			if got := len(download.GetContent()); got != 0 {
				t.Fatalf("event download content length = %d, want 0", got)
			}
			if download.GetCur() != expectedCur {
				t.Fatalf("event download cur = %d, want %d", download.GetCur(), expectedCur)
			}
			if download.GetSize() != uint64(fileSize) {
				t.Fatalf("event download size = %d, want %d", download.GetSize(), fileSize)
			}
			if download.GetChecksum() != checksum {
				t.Fatalf("event download checksum = %q, want %q", download.GetChecksum(), checksum)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("did not receive %s event for chunk %d", expectedOp, expectedCur)
		}
	}

	var lastResponse *implantpb.Spite
	for i := 1; i <= numChunks; i++ {
		chunk := make([]byte, chunkSize)
		resp := &implantpb.Spite{
			TaskId: task.Id,
			Body: &implantpb.Spite_DownloadResponse{
				DownloadResponse: &implantpb.DownloadResponse{
					Checksum: checksum,
					Cur:      int32(i),
					Size:     uint64(fileSize),
					Content:  chunk,
				},
			},
		}
		task.Done(resp, fmt.Sprintf("chunk %d/%d", i, numChunks))
		assertEvent(consts.CtrlTaskCallback, int32(i))

		if got := len(resp.GetDownloadResponse().GetContent()); got != chunkSize {
			t.Fatalf("source download content length = %d, want %d", got, chunkSize)
		}
		lastResponse = resp
	}

	task.Finish(lastResponse, "download completed")
	assertEvent(consts.CtrlTaskFinish, numChunks)
	if got := len(lastResponse.GetDownloadResponse().GetContent()); got != chunkSize {
		t.Fatalf("source download content length after finish = %d, want %d", got, chunkSize)
	}
}
