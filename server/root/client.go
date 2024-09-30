package root

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"google.golang.org/grpc"
)

func NewRootClient(addr string) (*RootClient, error) {
	ca, key, err := certutils.GetCertificateAuthority()
	if err != nil {
		return nil, err
	}
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKeyPEM := pem.EncodeToMemory(keyPEM)
	options, err := mtls.GetGrpcOptions(caCert, caCert, privateKeyPEM, certutils.RootName)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(addr, options...)
	if err != nil {
		return nil, err
	}

	return &RootClient{
		conn: conn,
		rpc:  clientrpc.NewRootRPCClient(conn),
	}, nil
}

type RootClient struct {
	conn *grpc.ClientConn
	rpc  clientrpc.RootRPCClient
}

func (client *RootClient) Execute(cmd Command, msg *rootpb.Operator) error {
	if len(msg.Args) == 0 && (msg.Op == "add" || msg.Op == "del") {
		fmt.Println("Name is required")
		return nil
	}

	_, err := cmd.Execute(client.rpc, msg)
	if err != nil {
		return err
	}
	return nil
}
