package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) GetGithubConfig(ctx context.Context, req *clientpb.Empty) (*clientpb.GithubWorkflowRequest, error) {
	githubConfig := configs.GetGithubConfig()
	if githubConfig == nil {
		return nil, errs.ErrNotFoundGithubConfig
	}
	return &clientpb.GithubWorkflowRequest{
		Owner:      githubConfig.Owner,
		Repo:       githubConfig.Repo,
		Token:      githubConfig.Token,
		WorkflowId: githubConfig.Workflow,
	}, nil
}

func (rpc *Server) UpdateGithubConfig(ctx context.Context, req *clientpb.GithubWorkflowRequest) (*clientpb.Empty, error) {
	err := configs.UpdateGithubConfig(&configs.GithubConfig{
		Owner:    req.Owner,
		Repo:     req.Repo,
		Token:    req.Token,
		Workflow: req.WorkflowId,
	})
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) GetNotifyConfig(ctx context.Context, req *clientpb.Empty) (*clientpb.Notify, error) {
	notifyConfig := configs.GetNotifyConfig()
	if notifyConfig == nil {
		return nil, errs.ErrNotFoundNotifyConfig
	}
	if notifyConfig.Telegram == nil {
		notifyConfig.Telegram = &configs.TelegramConfig{}
	}
	if notifyConfig.DingTalk == nil {
		notifyConfig.DingTalk = &configs.DingTalkConfig{}
	}
	if notifyConfig.Lark == nil {
		notifyConfig.Lark = &configs.LarkConfig{}
	}
	if notifyConfig.ServerChan == nil {
		notifyConfig.ServerChan = &configs.ServerChanConfig{}
	}
	return &clientpb.Notify{
		TelegramEnable:   notifyConfig.Telegram.Enable,
		TelegramApiKey:   notifyConfig.Telegram.APIKey,
		TelegramChatId:   notifyConfig.Telegram.ChatID,
		DingtalkEnable:   notifyConfig.DingTalk.Enable,
		DingtalkSecret:   notifyConfig.DingTalk.Secret,
		DingtalkToken:    notifyConfig.DingTalk.Token,
		LarkEnable:       notifyConfig.Lark.Enable,
		LarkWebhookUrl:   notifyConfig.Lark.WebHookUrl,
		ServerchanEnable: notifyConfig.ServerChan.Enable,
		ServerchanUrl:    notifyConfig.ServerChan.URL,
	}, nil
}

func (rpc *Server) UpdateNotifyConfig(ctx context.Context, req *clientpb.Notify) (*clientpb.Empty, error) {
	notifyConfig := &configs.NotifyConfig{
		Enable: req.TelegramEnable || req.DingtalkEnable || req.LarkEnable || req.ServerchanEnable,
		Telegram: &configs.TelegramConfig{
			Enable: req.TelegramEnable,
			APIKey: req.TelegramApiKey,
			ChatID: req.TelegramChatId,
		},
		DingTalk: &configs.DingTalkConfig{
			Enable: req.DingtalkEnable,
			Secret: req.DingtalkSecret,
			Token:  req.DingtalkToken,
		},
		Lark: &configs.LarkConfig{
			Enable:     req.LarkEnable,
			WebHookUrl: req.LarkWebhookUrl,
		},
		ServerChan: &configs.ServerChanConfig{
			Enable: req.ServerchanEnable,
			URL:    req.ServerchanUrl,
		},
	}
	err := configs.UpdateNotifyConfig(notifyConfig)
	if err != nil {
		return nil, err
	}
	err = core.EventBroker.InitService(notifyConfig)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RefreshConfig(ctx context.Context, req *clientpb.Empty) (*clientpb.Empty, error) {
	var server configs.ServerConfig
	err := configutil.LoadConfig(configs.CurrentServerConfigFilename, &server)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
