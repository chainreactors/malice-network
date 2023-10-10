package generate

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/utils/certs"
)

// initRootCA - Initialize the root CA
func initRootCA() {
	err := certs.InitRSACertificate("localhost", "root", true, false)
	if err != nil {
		logs.Log.Errorf("Failed to generate server certificate: %v", err)
	}
}

// InitClientCA - Initialize the client CA
func InitClientCA(host, user string) error {
	err := certs.InitRSACertificate(host, user, false, true)
	if err != nil {
		logs.Log.Errorf("Failed to generate client certificate: %v", err)
		return err
	}
	return nil
}
