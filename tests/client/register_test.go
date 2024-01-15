package client

import (
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestRegister(t *testing.T) {
	implant := common.NewImplant(common.DefaultListenerAddr, common.TestSid)
	implant.Register()
}
