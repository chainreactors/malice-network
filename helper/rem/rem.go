package rem

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/cryptography"
	rem "github.com/chainreactors/rem/core"
	remrunner "github.com/chainreactors/rem/runner"
	"net"
	"net/url"
)

func NewURL(schema, user, pwd, host, port string) *url.URL {
	var userinfo *url.Userinfo
	if pwd != "" && user != "" {
		userinfo = url.UserPassword(user, pwd)
	} else if user != "" {
		userinfo = url.User(user)
	}

	return &url.URL{
		User:   userinfo,
		Scheme: schema,
		Host:   net.JoinHostPort(host, port),
	}
}

func NewRemServer(conURL string) (*remrunner.Console, error) {
	u, err := rem.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option remrunner.Options
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
	console, err := remrunner.NewConsole(remRunner, remRunner.URLs)
	if err != nil {
		return nil, err
	}
	return console, nil
}

func NewRemClient(conURL string, remoteURL, localURL string) (*remrunner.Console, error) {
	u, err := rem.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option remrunner.Options
	err = option.ParseArgs([]string{"-c", conURL, "-r", remoteURL, "-l", localURL})
	if err != nil {
		return nil, err
	}
	remRunner, err := option.Prepare()
	if err != nil {
		return nil, err
	}
	remRunner.URLs.ConsoleURL = u
	console, err := remrunner.NewConsole(remRunner, remRunner.URLs)
	if err != nil {
		return nil, err
	}
	return console, nil
}
