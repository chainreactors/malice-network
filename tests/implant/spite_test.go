package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestReadSpites(t *testing.T) {
	client := common.NewClient(common.DefaultListenerAddr, "1234")
	msg, err := client.Read()

	fmt.Println(msg, err)
}
