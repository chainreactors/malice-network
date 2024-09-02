package root

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
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
	options, err := mtls.GetGrpcOptions(caCert, caCert, privateKeyPEM, certs.RootName)
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

	resp, err := cmd.Execute(client.rpc, msg)
	if err != nil {
		return err
	}
	if msg.Op == "add" {
		configDir, _ := os.Getwd()
		var conf *mtls.ClientConfig
		err := yaml.Unmarshal([]byte(resp.(*rootpb.Response).Response), &conf)
		if err != nil {
			return err
		}
		err = mtls.WriteConfig(conf, msg.Name, msg.Args[0])
		if err != nil {
			return err
		}
		yamlPath := filepath.Join(configDir, fmt.Sprintf("%s_%s.yaml", msg.Args[0], conf.LHost))
		logs.Log.Importantf("yaml file written to %s", yamlPath)
		return nil
	} else if msg.Op == "del" {
		logs.Log.Importantf("Client configuration removed from db")
		return nil
	}
	fmt.Println(resp)
	return nil
}
