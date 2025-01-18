package common

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"strconv"
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
		newConfirm := tui.NewConfirm(fmt.Sprintf("This command opsec value %d is too low, command will not execute. Are you sure you want to continue?", opsec))
		newModel := tui.NewModel(newConfirm, nil, false, true)
		err = newModel.Run()
		if err != nil {
			return err
		}
		if !newConfirm.Confirmed {
			return errors.New("operation cancelled by user")
		}
	}
	return nil
}
