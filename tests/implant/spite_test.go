package implant

import (
	"fmt"
	"github.com/chainreactors/malice-network/tests/common"
	"testing"
)

func TestReadSpites(t *testing.T) {
	client := common.NewImplant(common.DefaultListenerAddr, []byte{1, 2, 3, 4})
	msg, err := client.Read()

	fmt.Println(msg, err)
}
