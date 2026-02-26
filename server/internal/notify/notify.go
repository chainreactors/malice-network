package notify

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	golark "github.com/go-lark/lark"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/dingding"
	"github.com/nikoksr/notify/service/http"
	"github.com/nikoksr/notify/service/telegram"
)

type Notifier struct {
	notify *notify.Notify
	enable bool
}

func NewNotifier() Notifier {
	return Notifier{notify: notify.New(), enable: false}
}

func (n *Notifier) InitService(config *configs.NotifyConfig) error {
	if config == nil || !config.Enable {
		return nil
	}
	n.enable = true
	if config.Telegram != nil && config.Telegram.Enable {
		tg, err := telegram.New(config.Telegram.APIKey)
		if err != nil {
			return err
		}
		tg.SetParseMode(telegram.ModeMarkdown)
		tg.AddReceivers(config.Telegram.ChatID)
		n.notify.UseServices(tg)
	}
	if config.DingTalk != nil && config.DingTalk.Enable {
		dt := dingding.New(&dingding.Config{
			Token:  config.DingTalk.Token,
			Secret: config.DingTalk.Secret,
		})
		n.notify.UseServices(dt)
	}
	if config.Lark != nil && config.Lark.Enable {
		larkSvc := NewLarkWebhookService(config.Lark.WebHookUrl, config.Lark.Secret)
		n.notify.UseServices(larkSvc)
	}
	if config.ServerChan != nil && config.ServerChan.Enable {
		sc := http.New()
		sc.AddReceivers(&http.Webhook{
			URL:         config.ServerChan.URL,
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			BuildPayload: func(subject, message string) (payload any) {
				data := url.Values{}
				data.Set("subject", subject)
				data.Set("message", message)
				return data.Encode()
			},
		})
		n.notify.UseServices(sc)
	}
	if config.PushPlus != nil && config.PushPlus.Enable {
		pp := http.New()
		pp.AddReceivers(&http.Webhook{
			URL:         "https://www.pushplus.plus/send",
			Method:      "POST",
			ContentType: "application/json",
			BuildPayload: func(subject, message string) (payload any) {
				return map[string]string{
					"title":    subject,
					"content":  message,
					"token":    config.PushPlus.Token,
					"topic":    config.PushPlus.Topic,
					"channel":  config.PushPlus.Channel,
					"template": "markdown",
				}
			},
		})
		n.notify.UseServices(pp)
	}
	return nil
}

func (n *Notifier) Send(eventType, op, message string) {
	if !n.enable {
		return
	}
	title := fmt.Sprintf("[%s] %s", eventType, op)
	err := n.notify.Send(context.Background(), title, message)
	if err != nil {
		logs.Log.Errorf("Failed to send notification: %s", err)
	}
}

// LarkWebhookService implements notify.Notifier with optional signature verification.
type LarkWebhookService struct {
	bot    *golark.Bot
	secret string
}

func NewLarkWebhookService(webhookURL, secret string) *LarkWebhookService {
	return &LarkWebhookService{
		bot:    golark.NewNotificationBot(webhookURL),
		secret: secret,
	}
}

func (s *LarkWebhookService) Send(_ context.Context, subject, message string) error {
	content := golark.NewPostBuilder().
		Title(subject).
		TextTag(message, 1, false).
		Render()
	msg := golark.NewMsgBuffer(golark.MsgPost).Post(content)
	if s.secret != "" {
		msg.WithSign(s.secret, time.Now().Unix())
	}
	res, err := s.bot.PostNotificationV2(msg.Build())
	if err != nil {
		return fmt.Errorf("failed to post lark webhook message: %w", err)
	}
	if res.Code != 0 {
		return fmt.Errorf("lark send failed with code %d: %s", res.Code, res.StatusMessage)
	}
	return nil
}
