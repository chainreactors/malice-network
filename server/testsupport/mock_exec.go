//go:build mockimplant

package testsupport

import (
	"context"
	"time"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

type MockExecChunk struct {
	Delay  time.Duration
	Stdout []byte
	Stderr []byte
}

// SendRealisticExecStream mirrors the real implant's realtime exec shape:
// output chunks arrive first, then a final empty terminal marker closes the
// stream with end=true and the final status code.
func SendRealisticExecStream(
	ctx context.Context,
	send func(*implantpb.Spite) error,
	pid uint32,
	statusCode int32,
	chunks ...MockExecChunk,
) error {
	for _, chunk := range chunks {
		if chunk.Delay > 0 {
			select {
			case <-time.After(chunk.Delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if len(chunk.Stdout) == 0 && len(chunk.Stderr) == 0 {
			continue
		}
		if err := send(&implantpb.Spite{
			Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
				Pid:    pid,
				Stdout: append([]byte(nil), chunk.Stdout...),
				Stderr: append([]byte(nil), chunk.Stderr...),
				End:    false,
			}},
		}); err != nil {
			return err
		}
	}

	return send(&implantpb.Spite{
		Body: &implantpb.Spite_ExecResponse{ExecResponse: &implantpb.ExecResponse{
			Pid:        pid,
			StatusCode: statusCode,
			End:        true,
		}},
	})
}
