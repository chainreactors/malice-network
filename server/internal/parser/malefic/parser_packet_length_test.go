package malefic

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/gookit/config/v2"
)

func TestMaxPacketLenUsesParserValue(t *testing.T) {
	p := NewMaleficParser()
	p.MaxPacketLength = 4096

	got := p.maxPacketLen()
	if got != 4096 {
		t.Fatalf("maxPacketLen() = %d, want 4096", got)
	}
}

func TestMaxPacketLenFallsBackToGlobal(t *testing.T) {
	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	p := NewMaleficParser()
	// MaxPacketLength is 0 (unset)

	got := p.maxPacketLen()
	if got != 10485760 {
		t.Fatalf("maxPacketLen() = %d, want 10485760", got)
	}
}
