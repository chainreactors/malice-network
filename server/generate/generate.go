package generate

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/utils/certs"
	"os"
)

// InitRootCA - Initialize the root CA
func InitRootCA() {
	os.Mkdir(".config", 0744)
	os.Mkdir(".config/certs", 0744)
	_, _, err := certs.InitRSACertificate("localhost", "root", true, false)
	if err != nil {
		logs.Log.Errorf("Failed to generate server certificate: %v", err)
	}
}

// InitClientCA - Initialize the client CA
func InitClientCA(host, user string) ([]byte, []byte, error) {
	cert, key, err := certs.InitRSACertificate(host, user, false, true)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, err
}
