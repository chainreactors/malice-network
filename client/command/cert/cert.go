package cert

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"time"
)

func DeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	_, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	_, err = con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s delete success\n", certName)
	return nil
}

func UpdateCmd(cmd *cobra.Command, con *repl.Console) error {
	tls, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	certType, _ := cmd.Flags().GetString("type")
	_, err = con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
		Type: certType,
		Cert: tls.Cert.Cert,
		Key:  tls.Cert.Key,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert update %s success\n", certName)
	return nil
}

func GetCmd(cmd *cobra.Command, con *repl.Console) error {
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

func printCerts(certs *clientpb.Certs, con *repl.Console) {
	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Expire", "Expire", 25),
	}, true)

	for _, cert := range certs.Certs {
		_, notAfter, err := getCertExpireTime(cert.Cert)
		expireStr := ""
		if err == nil {
			expireStr = notAfter.Format("2006-01-02 15:04:05")
		}
		row := table.NewRow(table.RowData{
			"Name":   cert.Name,
			"Type":   cert.Type,
			"Expire": expireStr,
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
