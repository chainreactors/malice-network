package rem

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/rem/agent"
	rem "github.com/chainreactors/rem/protocol/core"
	remrunner "github.com/chainreactors/rem/runner"
	"github.com/chainreactors/rem/x/utils"
	"net"
	"net/url"
	"strconv"
)

func init() {
	utils.Log = logs.NewLogger(logs.InfoLevel)
}

func ParseRemCmd(args []string) (*remrunner.Options, error) {
	var option remrunner.Options
	err := option.ParseArgs(append([]string{"rem"}, args...))
	if err != nil {
		return nil, err
	}
	return &option, nil
}

func ParseConsole(conURL string) (*URL, error) {
	u, err := rem.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	return &URL{URL: u}, nil
}

type URL struct {
	*rem.URL
}

func (u *URL) Port() uint32 {
	port, _ := strconv.Atoi(u.URL.Port())
	return uint32(port)
}

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

type RemConsole struct {
	*remrunner.Console
}

func (rem *RemConsole) ToProtobuf() map[string]*clientpb.REMAgent {
	agents := make(map[string]*clientpb.REMAgent)
	agent.Agents.Range(func(key, value interface{}) bool {
		a := value.(*agent.Agent)
		agents[a.ID] = &clientpb.REMAgent{
			Id:     a.Name(),
			Mod:    a.Mod,
			Local:  a.LocalURL.String(),
			Remote: a.RemoteURL.String(),
		}
		return true
	})
	return agents
}

func NewRemServer(conURL string, ip string) (*RemConsole, error) {
	u, err := rem.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option remrunner.Options
	var args []string
	if ip == "" {
		args = []string{"rem", "-c", conURL}
	} else {
		args = []string{"rem", "-c", conURL, "-i", ip}
	}
	err = option.ParseArgs(args)
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
	return &RemConsole{console}, nil
}

func NewRemClient(conURL string, args []string) (*RemConsole, error) {
	u, err := rem.NewConsoleURL(conURL)
	if err != nil {
		return nil, err
	}
	var option remrunner.Options
	err = option.ParseArgs(args)
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
	return &RemConsole{console}, nil
}
