package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TestAddCredentialResolvesTaskWithoutSessionEnvelope(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-credential", "rpc-context-credential-pipe", true)
	task := seedRPCTestTask(t, sess, "credential")

	_, err := (&Server{}).AddCredential(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "nonce-cred",
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "server.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddCredential failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "nonce-cred",
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
	if contexts.Contexts[0].Nonce != "nonce-cred" {
		t.Fatalf("context nonce = %q, want nonce-cred", contexts.Contexts[0].Nonce)
	}
	if contexts.Contexts[0].Session == nil || contexts.Contexts[0].Session.SessionId != sess.ID {
		t.Fatalf("context session = %#v, want session %s", contexts.Contexts[0].Session, sess.ID)
	}
}

func TestAddContextPublishesLifecycleEvent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-add-event", "rpc-context-add-event-pipe", true)
	task := seedRPCTestTask(t, sess, "credential")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).AddContext(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "generic-context-event",
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "generic.local",
		}),
	}); err != nil {
		t.Fatalf("AddContext failed: %v", err)
	}

	event := waitForLifecycleEvent(t, events, consts.ContextCredential)
	if event.EventType != consts.EventContext || !event.Important {
		t.Fatalf("AddContext event = %#v, want important context event", event)
	}
}

func TestAddDownloadPublishesLifecycleEvent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-download-event", "rpc-context-download-event-pipe", true)
	task := seedRPCTestTask(t, sess, "download")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).AddDownload(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextDownload,
		Nonce: "download-context-event",
		Value: output.MarshalContext(&output.DownloadContext{FileDescriptor: &output.FileDescriptor{
			Name:       "download.bin",
			TargetPath: "C:/download.bin",
			FilePath:   "/tmp/download.bin",
			Size:       8,
		}}),
	}); err != nil {
		t.Fatalf("AddDownload failed: %v", err)
	}

	event := waitForLifecycleEvent(t, events, consts.ContextDownload)
	if event.EventType != consts.EventContext || !event.Important {
		t.Fatalf("AddDownload event = %#v, want important context event", event)
	}
}

func TestDeleteContextPublishesLifecycleEvent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-delete-event", "rpc-context-delete-event-pipe", true)
	task := seedRPCTestTask(t, sess, "credential")
	contextModel, err := db.SaveContext(&clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "delete-context-event",
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "delete.local",
		}),
	})
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).DeleteContext(context.Background(), &clientpb.Context{Id: contextModel.ID.String()}); err != nil {
		t.Fatalf("DeleteContext failed: %v", err)
	}

	event := waitForLifecycleEvent(t, events, consts.CtrlContextDelete)
	if event.EventType != consts.EventContext || !event.Important {
		t.Fatalf("DeleteContext event = %#v, want important context event", event)
	}
}

func TestSaveTaskContextsPublishesReplayableLifecycleEvent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-event", "rpc-context-event-pipe", true)
	task := seedRPCTestTask(t, sess, "port")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	saveTaskContextsFromContent(task, contextRequestMeta{
		ContextType: output.GOGOPortType,
		Nonce:       "port-event",
	}, []byte(`{"ip":"127.0.0.1","port":"8080","protocol":"tcp","status":"open"}`))

	event := waitForLifecycleEvent(t, events, consts.ContextPort)
	if event.EventType != consts.EventContext || !event.Important {
		t.Fatalf("context lifecycle event = %#v, want important context event", event)
	}
}

func TestCompletedFileContextPublishesReplayableLifecycleEvent(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-file-context-event", "rpc-file-context-event-pipe", true)
	task := seedRPCTestTask(t, sess, "download")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	const fileID = uint32(7)
	openPayload := make([]byte, 8+len("result.bin"))
	binary.LittleEndian.PutUint32(openPayload[:4], fileID)
	copy(openPayload[8:], "result.bin")
	if err := core.HandleFileOperations("open", openPayload, task); err != nil {
		t.Fatalf("open file context failed: %v", err)
	}
	writePayload := make([]byte, 4+len("result"))
	binary.LittleEndian.PutUint32(writePayload[:4], fileID)
	copy(writePayload[4:], "result")
	if err := core.HandleFileOperations("write", writePayload, task); err != nil {
		t.Fatalf("write file context failed: %v", err)
	}
	closePayload := make([]byte, 4)
	binary.LittleEndian.PutUint32(closePayload, fileID)
	if err := core.HandleFileOperations("close", closePayload, task); err != nil {
		t.Fatalf("close file context failed: %v", err)
	}

	event := waitForLifecycleEvent(t, events, consts.CtrlContextFileClose)
	if event.EventType != consts.EventContext || !event.Important {
		t.Fatalf("file context lifecycle event = %#v, want important context event", event)
	}
}

func TestAddScreenShotAcceptsRawContentWithoutMetadataValue(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-screenshot", "rpc-context-screenshot-pipe", true)
	task := seedRPCTestTask(t, sess, "screenshot")

	payload := append([]byte{0, 0, 0, 0}, []byte("jpeg-data")...)
	if _, err := (&Server{}).AddScreenShot(context.Background(), &clientpb.Context{
		Task:    task.ToProtobuf(),
		Content: payload,
		Type:    consts.ContextScreenShot,
	}); err != nil {
		t.Fatalf("AddScreenShot failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextScreenShot,
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
	screenshotCtx, err := output.ToContext[*output.ScreenShotContext](contexts.Contexts[0])
	if err != nil {
		t.Fatalf("ToContext failed: %v", err)
	}
	content, err := os.ReadFile(screenshotCtx.FilePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "jpeg-data" {
		t.Fatalf("screenshot file content = %q, want jpeg-data", content)
	}
}

func TestAddUploadCreatesContextRecord(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-upload", "rpc-context-upload-pipe", true)
	task := seedRPCTestTask(t, sess, "upload")

	if _, err := (&Server{}).AddUpload(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextUpload,
		Value: output.MarshalContext(&output.UploadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "upload.bin",
				TargetPath: "C:\\temp\\upload.bin",
				Size:       11,
			},
		}),
	}); err != nil {
		t.Fatalf("AddUpload failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextUpload,
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
}

func TestSyncRequiresIdentifier(t *testing.T) {
	if _, err := (&Server{}).Sync(context.Background(), &clientpb.Sync{}); err == nil || err.Error() != "context id or task id is required" {
		t.Fatalf("Sync error = %v, want context id or task id is required", err)
	}
}

func TestSyncReturnsContextWithoutContentWhenBackingFileIsMissing(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-sync", "rpc-context-sync-pipe", true)
	task := seedRPCTestTask(t, sess, "download")

	model, err := db.SaveContext(&clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    consts.ContextDownload,
		Value: output.MarshalContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "missing.bin",
				TargetPath: "remote/missing.bin",
				FilePath:   "Z:/definitely-missing.bin",
				Size:       3,
			},
		}),
	})
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}

	ctx, err := (&Server{}).Sync(context.Background(), &clientpb.Sync{ContextId: model.ID.String()})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if ctx.Id != model.ID.String() {
		t.Fatalf("context id = %q, want %q", ctx.Id, model.ID.String())
	}
	if len(ctx.Content) != 0 {
		t.Fatalf("context content length = %d, want 0", len(ctx.Content))
	}
}

func TestSyncStreamStreamsContextFileInChunks(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-sync-stream", "rpc-context-sync-stream-pipe", true)
	task := seedRPCTestTask(t, sess, "download")

	content := bytes.Repeat([]byte("x"), int(syncStreamChunkSize)+17)
	filePath := filepath.Join(t.TempDir(), "download.bin")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	model, err := db.SaveContext(&clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    consts.ContextDownload,
		Value: output.MarshalContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "download.bin",
				TargetPath: "remote/download.bin",
				FilePath:   filePath,
				Size:       int64(len(content)),
			},
		}),
	})
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}

	stream := newCaptureContextChunkStream(context.Background())
	if err := (&Server{}).SyncStream(&clientpb.Sync{ContextId: model.ID.String()}, stream); err != nil {
		t.Fatalf("SyncStream failed: %v", err)
	}

	if len(stream.chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(stream.chunks))
	}
	if stream.chunks[0].Header == nil || stream.chunks[0].Header.Id != model.ID.String() {
		t.Fatalf("header = %#v, want context id %s", stream.chunks[0].Header, model.ID.String())
	}
	if len(stream.chunks[0].Header.Content) != 0 {
		t.Fatalf("header content length = %d, want 0", len(stream.chunks[0].Header.Content))
	}
	if stream.chunks[1].Offset != 0 {
		t.Fatalf("first content offset = %d, want 0", stream.chunks[1].Offset)
	}
	if got := len(stream.chunks[1].Content); got != 512*1024 {
		t.Fatalf("first content length = %d, want %d", got, 512*1024)
	}
	if stream.chunks[2].Offset != syncStreamChunkSize {
		t.Fatalf("second content offset = %d, want %d", stream.chunks[2].Offset, syncStreamChunkSize)
	}
	combined := append([]byte{}, stream.chunks[1].Content...)
	combined = append(combined, stream.chunks[2].Content...)
	if !bytes.Equal(combined, content) {
		t.Fatalf("streamed content mismatch")
	}
	if !stream.chunks[2].Eof {
		t.Fatalf("last content chunk eof = false, want true")
	}
}

func TestSendContextContentStreamSendsBeforeReaderEOF(t *testing.T) {
	reader := &blockingContextReader{
		first:   []byte("abc"),
		rest:    []byte("def"),
		release: make(chan struct{}),
	}
	stream := newCaptureContextChunkStream(context.Background())
	stream.contentSent = make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- sendContextContentStream(&clientpb.Context{Id: "streaming"}, 6, reader, stream)
	}()

	select {
	case <-stream.contentSent:
	case <-time.After(time.Second):
		t.Fatal("first content chunk was not sent before reader EOF")
	}

	if len(stream.chunks) != 2 {
		t.Fatalf("chunk count before EOF = %d, want 2", len(stream.chunks))
	}
	if got := string(stream.chunks[1].Content); got != "abc" {
		t.Fatalf("first content chunk = %q, want abc", got)
	}

	close(reader.release)

	if err := <-errCh; err != nil {
		t.Fatalf("sendContextContentStream failed: %v", err)
	}
	if len(stream.chunks) != 3 {
		t.Fatalf("final chunk count = %d, want 3", len(stream.chunks))
	}
	combined := append([]byte{}, stream.chunks[1].Content...)
	combined = append(combined, stream.chunks[2].Content...)
	if string(combined) != "abcdef" {
		t.Fatalf("combined content = %q, want abcdef", combined)
	}
	if !stream.chunks[2].Eof {
		t.Fatalf("last content chunk eof = false, want true")
	}
}

func TestSendContextContentStreamRejectsTruncatedContent(t *testing.T) {
	stream := newCaptureContextChunkStream(context.Background())
	err := sendContextContentStream(
		&clientpb.Context{Id: "truncated"},
		6,
		bytes.NewReader([]byte("abc")),
		stream,
	)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want io.ErrUnexpectedEOF", err)
	}
	if len(stream.chunks) != 2 || stream.chunks[0].Header == nil {
		t.Fatalf("chunks = %#v, want metadata header and partial content", stream.chunks)
	}
	if got := string(stream.chunks[1].Content); got != "abc" || stream.chunks[1].Eof {
		t.Fatalf("partial chunk = %#v, want abc with eof=false", stream.chunks[1])
	}
}

func seedRPCTestTask(t testing.TB, sess *core.Session, taskType string) *core.Task {
	t.Helper()

	task := sess.NewTask(taskType, 1)
	task.Cur = 1
	task.CreatedAt = time.Now()
	task.FinishedAt = task.CreatedAt
	task.CallBy = consts.CalleeCMD

	if err := db.AddTask(task.ToProtobuf()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := db.UpdateTaskFinish(task.TaskID()); err != nil {
		t.Fatalf("UpdateTaskFinish failed: %v", err)
	}
	return task
}

type captureContextChunkStream struct {
	ctx         context.Context
	chunks      []*clientpb.ContextChunk
	contentSent chan struct{}
	once        sync.Once
}

func newCaptureContextChunkStream(ctx context.Context) *captureContextChunkStream {
	return &captureContextChunkStream{ctx: ctx}
}

func (s *captureContextChunkStream) Send(chunk *clientpb.ContextChunk) error {
	copied := proto.Clone(chunk).(*clientpb.ContextChunk)
	s.chunks = append(s.chunks, copied)
	if len(chunk.Content) > 0 && s.contentSent != nil {
		s.once.Do(func() { close(s.contentSent) })
	}
	return nil
}

func (s *captureContextChunkStream) SetHeader(metadata.MD) error {
	return nil
}

func (s *captureContextChunkStream) SendHeader(metadata.MD) error {
	return nil
}

func (s *captureContextChunkStream) SetTrailer(metadata.MD) {}

func (s *captureContextChunkStream) Context() context.Context {
	return s.ctx
}

func (s *captureContextChunkStream) SendMsg(any) error {
	return nil
}

func (s *captureContextChunkStream) RecvMsg(any) error {
	return io.EOF
}

type blockingContextReader struct {
	first   []byte
	rest    []byte
	release chan struct{}
	reads   int
}

func (r *blockingContextReader) Read(p []byte) (int, error) {
	switch r.reads {
	case 0:
		r.reads++
		return copy(p, r.first), nil
	case 1:
		r.reads++
		<-r.release
		return copy(p, r.rest), io.EOF
	default:
		return 0, io.EOF
	}
}
