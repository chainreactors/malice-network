package common

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func OpsecConfirm(cmd *cobra.Command) error {
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
		yes, _ := cmd.Flags().GetBool("yes")
		if yes {
			return nil
		}
		newConfirm := tui.NewConfirm(fmt.Sprintf("This command opsec value %.1f is too low, command will not execute. Are you sure you want to continue?", opsec))
		err = newConfirm.Run()
		if err != nil {
			return err
		}
		if !newConfirm.GetConfirmed() {
			return errors.New("operation cancelled by user")
		}
	}
	return nil
}
