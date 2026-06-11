package common

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func OpsecConfirm(cmd *cobra.Command, con *core.Console) error {
	opsec, err := strconv.ParseFloat(cmd.Annotations["opsec"], 64)
	if err != nil {
		return err
	}
	setting, err := assets.GetSetting()
	if err != nil {
		return err
	}
	threshold := setting.OpsecThreshold
	if err != nil {
		return err
	}
	if opsec < threshold {
		confirmed, err := Confirm(cmd, con, fmt.Sprintf("This command opsec value %.1f is too low, command will not execute. Are you sure you want to continue?", opsec))
		if err != nil {
			return err
		}
		if !confirmed {
			return errors.New("operation cancelled by user")
		}
	}
	return nil
}
