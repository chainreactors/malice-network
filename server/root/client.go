package root

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"google.golang.org/grpc"
)

func NewRootClient(addr string) (*RootClient, error) {
	ca, key, err := certs.GetCertificateAuthority()
	if err != nil {
		return nil, err
	}
	caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKeyPEM := pem.EncodeToMemory(keyPEM)
	options, err := mtls.GetGrpcOptions(string(caCert), string(caCert), string(privateKeyPEM), certs.RootName)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(addr, options...)
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
	resp, err := cmd.Execute(client.rpc, msg)
	if err != nil {
		return err
	}
	if msg.Op == "add" {
		err = mtls.WriteConfig(resp.(*rootpb.Response).Response, msg.Name, msg.Args[0])
		if err != nil {
			return err
		}
		fmt.Println("Client configuration written to disk")
		return nil
	} else if msg.Op == "del" {
		fmt.Println("Client configuration removed from db")
		return nil
	}
	fmt.Println(resp)
	return nil
}
