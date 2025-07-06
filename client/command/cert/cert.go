package cert

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"time"
)

var (
	certFile = "cert.pem"
	keyFile  = "key.pem"
	caFile   = "ca-cert.pem"
)

func DeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	certName := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s delete success\n", certName)
	return nil
}

func UpdateCmd(cmd *cobra.Command, con *repl.Console) error {
	certName := cmd.Flags().Arg(0)
	certPath, _ := cmd.Flags().GetString("cert")
	keyPath, _ := cmd.Flags().GetString("key")
	certType, _ := cmd.Flags().GetString("type")
	var cert, key string
	var err error
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			return err
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			return err
		}
	}
	_, err = con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
		Type: certType,
		Cert: cert,
		Key:  key,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert update %s success\n", certName)
	return nil
}

func DownloadCmd(cmd *cobra.Command, con *repl.Console) error {
	certName := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	cert, err := con.Rpc.DownloadCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
	})
	if err != nil {
		return nil
	}
	printCert(cert)
	var path string
	if output != "" {
		path = filepath.Join(assets.GetTempDir(), output)
	} else {
		path = filepath.Join(assets.GetTempDir(), certName)
	}
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(filepath.Join(path, certFile), []byte(cert.Cert.Cert))
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(filepath.Join(path, keyFile), []byte(cert.Cert.Key))
	if err != nil {
		return err
	}
	if cert.Ca.Cert != "" {
		err = certs.SaveToPEMFile(filepath.Join(path, caFile), []byte(cert.Ca.Cert))
		if err != nil {
			return err
		}
	}
	con.Log.Infof("cert save in %s\n", path)
	return nil
}

func GetCertCmd(cmd *cobra.Command, con *repl.Console) error {
	certs, err := con.Rpc.GetAllCertificates(con.Context(), &clientpb.Empty{})
	if err != nil {
		return nil
	}
	if len(certs.Certs) > 0 {
		printCerts(certs, con)
	} else {
		con.Log.Infof("no cert\n")
	}
	return nil
}

func printCert(cert *clientpb.TLS) {
	_, notAfter, err := getCertExpireTime(cert.Cert.Cert)
	expireStr := ""
	if err == nil {
		expireStr = notAfter.Format("2006-01-02 15:04:05")
	}
	certMap := map[string]interface{}{
		"Name":               cert.Cert.Name,
		"Type":               cert.Cert.Type,
		"Organization":       cert.CertSubject.O,
		"Country":            cert.CertSubject.C,
		"Locality":           cert.CertSubject.L,
		"OrganizationalUnit": cert.CertSubject.Ou,
		"StreetAddress":      cert.CertSubject.St,
		"Expire":             expireStr,
	}

	tui.RenderKV(certMap)
}

func printCerts(certs *clientpb.Certs, con *repl.Console) {
	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Organization", "Organization", 20),
		table.NewColumn("Country", "Country", 20),
		table.NewColumn("Locality", "Locality", 20),
		table.NewColumn("OrganizationalUnit", "OrganizationalUnit", 30),
		table.NewColumn("StreetAddress", "StreetAddress", 20),
		table.NewColumn("Expire", "Expire", 25),
	}, true)

	for _, cert := range certs.Certs {
		_, notAfter, err := getCertExpireTime(cert.Cert.Cert)
		expireStr := ""
		if err == nil {
			expireStr = notAfter.Format("2006-01-02 15:04:05")
		}
		row := table.NewRow(table.RowData{
			"Name":               cert.Cert.Name,
			"Type":               cert.Cert.Type,
			"Organization":       cert.CertSubject.O,
			"Country":            cert.CertSubject.C,
			"Locality":           cert.CertSubject.L,
			"OrganizationalUnit": cert.CertSubject.Ou,
			"StreetAddress":      cert.CertSubject.St,
			"Expire":             expireStr,
		})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}

func getCertExpireTime(certPEM string) (notBefore, notAfter time.Time, err error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		err = errors.New("failed to parse certificate PEM")
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return
	}
	return cert.NotBefore, cert.NotAfter, nil
}
