package core

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

type Pipeline interface {
	ID() string
	Start() error
	Close() error
	ToProtobuf() *clientpb.Pipeline
}

type Pipelines struct {
	mu sync.RWMutex
	m  map[string]Pipeline
}

func NewPipelines() Pipelines {
	return Pipelines{m: make(map[string]Pipeline)}
}

func (ps *Pipelines) Add(p Pipeline) {
	ps.mu.Lock()
	ps.m[p.ID()] = p
	ps.mu.Unlock()
}

func (ps *Pipelines) Get(id string) Pipeline {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.m[id]
}

func (ps *Pipelines) Delete(id string) {
	ps.mu.Lock()
	delete(ps.m, id)
	ps.mu.Unlock()
}

func (ps *Pipelines) ToProtobuf() *clientpb.Pipelines {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	var pls = &clientpb.Pipelines{
		Pipelines: make([]*clientpb.Pipeline, 0, len(ps.m)),
	}
	for _, p := range ps.m {
		pls.Pipelines = append(pls.Pipelines, p.ToProtobuf())
	}
	return pls
}

func FromPipeline(pipeline *clientpb.Pipeline) *PipelineConfig {
	return &PipelineConfig{
		ListenerID:   pipeline.ListenerId,
		Parser:       pipeline.Parser,
		TLSConfig:    implanttypes.FromTls(pipeline.Tls),
		Encryption:   implanttypes.FromEncryptions(pipeline.GetEncryption()),
		SecureConfig: implanttypes.FromSecure(pipeline.Secure),
	}
}

type PipelineConfig struct {
	ListenerID   string
	Parser       string
	TLSConfig    *implanttypes.TlsConfig
	Encryption   implanttypes.EncryptionsConfig
	SecureConfig *implanttypes.SecureConfig
}

func (p *PipelineConfig) WrapConn(conn io.ReadWriteCloser) (*cryptostream.Conn, error) {
	if p == nil {
		return nil, errors.New("pipeline config is nil")
	}
	crys, err := configs.NewCrypto(p.Encryption.ToProtobuf())
	if err != nil {
		return nil, err
	}
	return cryptostream.WrapPeekConn(conn, crys, p.Parser)
}

// WrapBindConn wraps a connection for bind mode without pre-reading
// Bind mode expects server to send data first, then receive response
func (p *PipelineConfig) WrapBindConn(conn io.ReadWriteCloser) (*cryptostream.Conn, error) {
	if p == nil {
		return nil, errors.New("pipeline config is nil")
	}
	crys, err := configs.NewCrypto(p.Encryption.ToProtobuf())
	if err != nil {
		return nil, err
	}
	return cryptostream.WrapBindConn(conn, crys)
}

// PipelineRuntimeErrorHandler builds a standard error handler for pipeline
// runtime goroutines. All pipeline types (tcp, http, bind, rem, webshell) share
// the same pattern: log the error, disable the pipeline, optionally run cleanup,
// and publish an event.
func PipelineRuntimeErrorHandler(typeName, pipelineName, listenerID string, disabler func(), cleanup func(), op ...string) GoErrorHandler {
	label := fmt.Sprintf("%s pipeline %s", typeName, pipelineName)
	ctrlOp := consts.CtrlPipelineStop
	if len(op) > 0 {
		ctrlOp = op[0]
	}
	return CombineErrorHandlers(
		LogGuardedError(label),
		func(err error) {
			disabler()
			if cleanup != nil {
				cleanup()
			}
			if EventBroker != nil {
				EventBroker.Publish(Event{
					EventType: consts.EventListener,
					Op:        ctrlOp,
					Listener:  &clientpb.Listener{Id: listenerID},
					Message:   label,
					Err:       ErrorText(err),
					Important: true,
				})
			}
		},
	)
}
