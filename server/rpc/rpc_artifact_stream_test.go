package rpc

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TestDownloadArtifactStreamStreamsOriginalFileInChunks(t *testing.T) {
	newRPCTestEnv(t)
	content := bytes.Repeat([]byte("x"), 512*1024+17)
	artifact, err := db.SaveUploadedArtifact(&clientpb.Artifact{
		Name:     "stream-original",
		Type:     "beacon",
		Platform: "windows",
		Format:   ".exe",
	})
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(artifact.Path), 0o700); err != nil {
		t.Fatalf("MkdirAll artifact path: %v", err)
	}
	if err := os.WriteFile(artifact.Path, content, 0o600); err != nil {
		t.Fatalf("WriteFile artifact: %v", err)
	}

	stream := newCaptureArtifactChunkStream(context.Background())
	if err := (&Server{}).DownloadArtifactStream(&clientpb.Artifact{
		Name:   artifact.Name,
		Format: consts.FormatExecutable,
	}, stream); err != nil {
		t.Fatalf("DownloadArtifactStream: %v", err)
	}

	if len(stream.chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(stream.chunks))
	}
	header := stream.chunks[0]
	if header.Header == nil || header.Header.Name != artifact.Name {
		t.Fatalf("header = %#v, want artifact %q", header.Header, artifact.Name)
	}
	if len(header.Header.Bin) != 0 || len(header.Content) != 0 {
		t.Fatal("stream header must not contain artifact bytes")
	}
	if header.TotalSize != int64(len(content)) || header.Eof {
		t.Fatalf("header total/eof = %d/%v, want %d/false", header.TotalSize, header.Eof, len(content))
	}
	if stream.chunks[1].Offset != 0 || stream.chunks[2].Offset != 512*1024 {
		t.Fatalf("content offsets = %d, %d", stream.chunks[1].Offset, stream.chunks[2].Offset)
	}
	if !stream.chunks[2].Eof {
		t.Fatal("last artifact chunk must set eof")
	}
	if got := collectArtifactContent(stream.chunks); !bytes.Equal(got, content) {
		t.Fatal("streamed artifact content mismatch")
	}
}

func TestSendArtifactContentStreamSendsBeforeReaderEOF(t *testing.T) {
	reader := &blockingContextReader{
		first:   []byte("abc"),
		rest:    []byte("def"),
		release: make(chan struct{}),
	}
	stream := newCaptureArtifactChunkStream(context.Background())
	stream.contentSent = make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- sendArtifactContentStream(&clientpb.Artifact{Name: "streaming"}, 6, reader, stream)
	}()

	select {
	case <-stream.contentSent:
	case <-time.After(time.Second):
		t.Fatal("first artifact chunk was not sent before reader EOF")
	}
	if got := string(stream.chunks[1].Content); got != "abc" {
		t.Fatalf("first artifact content = %q, want abc", got)
	}

	close(reader.release)
	if err := <-errCh; err != nil {
		t.Fatalf("sendArtifactContentStream: %v", err)
	}
	if got := collectArtifactContent(stream.chunks); string(got) != "abcdef" {
		t.Fatalf("streamed content = %q, want abcdef", got)
	}
}

func TestDownloadArtifactStreamPreservesConvertedOutput(t *testing.T) {
	newRPCTestEnv(t)
	content := []byte{0x00, 0x41, 0xff}
	artifact, err := db.SaveUploadedArtifact(&clientpb.Artifact{
		Name:     "stream-converted",
		Type:     "beacon",
		Platform: "linux",
		Format:   "",
	})
	if err != nil {
		t.Fatalf("SaveUploadedArtifact: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(artifact.Path), 0o700); err != nil {
		t.Fatalf("MkdirAll artifact path: %v", err)
	}
	if err := os.WriteFile(artifact.Path, content, 0o600); err != nil {
		t.Fatalf("WriteFile artifact: %v", err)
	}

	request := &clientpb.Artifact{Name: artifact.Name, Format: consts.FormatHexOneLine}
	want, err := (&Server{}).DownloadArtifact(context.Background(), request)
	if err != nil {
		t.Fatalf("DownloadArtifact: %v", err)
	}
	stream := newCaptureArtifactChunkStream(context.Background())
	if err := (&Server{}).DownloadArtifactStream(request, stream); err != nil {
		t.Fatalf("DownloadArtifactStream: %v", err)
	}
	if got := collectArtifactContent(stream.chunks); !bytes.Equal(got, want.Bin) {
		t.Fatalf("converted stream = %q, want %q", got, want.Bin)
	}
	if stream.chunks[0].Header.GetFormat() != want.Format {
		t.Fatalf("stream format = %q, want %q", stream.chunks[0].Header.GetFormat(), want.Format)
	}
}

func collectArtifactContent(chunks []*clientpb.ArtifactChunk) []byte {
	var content []byte
	for _, chunk := range chunks {
		content = append(content, chunk.Content...)
	}
	return content
}

type captureArtifactChunkStream struct {
	ctx         context.Context
	chunks      []*clientpb.ArtifactChunk
	contentSent chan struct{}
	once        sync.Once
}

func newCaptureArtifactChunkStream(ctx context.Context) *captureArtifactChunkStream {
	return &captureArtifactChunkStream{ctx: ctx}
}

func (s *captureArtifactChunkStream) Send(chunk *clientpb.ArtifactChunk) error {
	s.chunks = append(s.chunks, proto.Clone(chunk).(*clientpb.ArtifactChunk))
	if len(chunk.Content) > 0 && s.contentSent != nil {
		s.once.Do(func() { close(s.contentSent) })
	}
	return nil
}

func (s *captureArtifactChunkStream) SetHeader(metadata.MD) error  { return nil }
func (s *captureArtifactChunkStream) SendHeader(metadata.MD) error { return nil }
func (s *captureArtifactChunkStream) SetTrailer(metadata.MD)       {}
func (s *captureArtifactChunkStream) Context() context.Context     { return s.ctx }
func (s *captureArtifactChunkStream) SendMsg(any) error            { return nil }
func (s *captureArtifactChunkStream) RecvMsg(any) error            { return io.EOF }
