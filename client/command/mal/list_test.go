package mal

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestListMalManifestInitializesManager(t *testing.T) {
	con := newMalTestConsole(t, false)
	cmd := &cobra.Command{Use: "list"}

	ListMalManifest(cmd, con)

	if con.MalManager == nil {
		t.Fatal("expected mal manager to be initialized")
	}
}
