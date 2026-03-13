package listener

import (
	"net/http/httptest"
	"testing"
)

func TestHTTPPipelineHandlerRecoversAndWritesInternalServerError(t *testing.T) {
	pipeline := &HTTPPipeline{Name: "http-a"}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	resp := httptest.NewRecorder()

	pipeline.handler(resp, req)

	if resp.Code != 500 {
		t.Fatalf("status code = %d, want 500", resp.Code)
	}
}
