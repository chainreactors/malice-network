package proxy

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/cryptography"
	rem "github.com/chainreactors/rem/runner"
	"github.com/chainreactors/rem/utils"
)

func NewRemServer(conURL string) (*rem.Console, error) {
	u, err := utils.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option rem.Options
	err = option.ParseArgs([]string{"-c", conURL})
	if err != nil {
		return nil, err
	}
	remRunner, err := option.Prepare()
	if err != nil {
		return nil, err
	}
	remRunner.URLs.ConsoleURL = u
	remRunner.Subscribe = fmt.Sprintf("http://0.0.0.0:%d", cryptography.RandomInRange(20000, 65500))
	console, err := rem.NewConsole(remRunner, remRunner.URLs)
	if err != nil {
		return nil, err
	}
	return console, nil
}

func NewRemClient(conURL string, remoteURL, localURL string) (*rem.Console, error) {
	u, err := utils.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option rem.Options
	err = option.ParseArgs([]string{"-c", conURL, "-r", remoteURL, "-l", localURL})
	if err != nil {
		return nil, err
	}
	remRunner, err := option.Prepare()
	if err != nil {
		return nil, err
	}
	remRunner.URLs.ConsoleURL = u
	console, err := rem.NewConsole(remRunner, remRunner.URLs)
	if err != nil {
		return nil, err
	}
	return console, nil
}
