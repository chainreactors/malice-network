package notify

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	golark "github.com/go-lark/lark"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/dingding"
	"github.com/nikoksr/notify/service/http"
	"github.com/nikoksr/notify/service/telegram"
)

// Notifier publishes complete, immutable service sets: a replacement is built
// and validated in memory first, then atomically swapped in, so a failed
// rebuild never touches the running services and a send never observes a
// half-initialized set. A Notifier must not be copied after first use.
type Notifier struct {
	active atomic.Pointer[notify.Notify]
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

// buildServices validates cfg and constructs a complete replacement service
// set without touching the currently active one.
func buildServices(config *configs.NotifyConfig) (*notify.Notify, error) {
	next := notify.New()
	if config == nil || !config.Enable {
		next.Disabled = true
		return next, nil
	}
	// Validate every enabled channel before constructing any provider, so a
	// misconfiguration fails the whole update instead of partially building it.
	if config.Telegram != nil && config.Telegram.Enable &&
		(strings.TrimSpace(config.Telegram.APIKey) == "" || config.Telegram.ChatID == 0) {
		return nil, errors.New("server.notify.telegram requires api_key and a nonzero chat_id")
	}
	if config.DingTalk != nil && config.DingTalk.Enable && strings.TrimSpace(config.DingTalk.Token) == "" {
		return nil, errors.New("server.notify.dingtalk requires token")
	}
	if config.Lark != nil && config.Lark.Enable {
		if err := validateWebhook("server.notify.lark.webhook_url", config.Lark.WebHookUrl); err != nil {
			return nil, err
		}
	}
	if config.ServerChan != nil && config.ServerChan.Enable {
		if err := validateWebhook("server.notify.serverchan.url", config.ServerChan.URL); err != nil {
			return nil, err
		}
	}
	if config.PushPlus != nil && config.PushPlus.Enable && strings.TrimSpace(config.PushPlus.Token) == "" {
		return nil, errors.New("server.notify.pushplus requires token")
	}
	if config.Telegram != nil && config.Telegram.Enable {
		tg, err := telegram.New(config.Telegram.APIKey)
		if err != nil {
			return nil, err
		}
		tg.SetParseMode(telegram.ModeMarkdown)
		tg.AddReceivers(config.Telegram.ChatID)
		next.UseServices(tg)
	}
	if config.DingTalk != nil && config.DingTalk.Enable {
		dt := dingding.New(&dingding.Config{
			Token:  config.DingTalk.Token,
			Secret: config.DingTalk.Secret,
		})
		next.UseServices(dt)
	}
	if config.Lark != nil && config.Lark.Enable {
		larkSvc := NewLarkWebhookService(config.Lark.WebHookUrl, config.Lark.Secret)
		next.UseServices(larkSvc)
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
		next.UseServices(sc)
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
		next.UseServices(pp)
	}
	return next, nil
}

func validateWebhook(path, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s: expected an absolute http(s) url", path)
	}
	return nil
}

// Prepare builds and validates the replacement service set for config without
// changing the active one. The returned commit activates the replacement
// atomically and cannot fail; callers that persist the configuration first
// invoke commit only after the write succeeds, keeping the persisted
// configuration and the running service in lockstep.
func (n *Notifier) Prepare(config *configs.NotifyConfig) (commit func(), err error) {
	if n == nil {
		return nil, errors.New("notifier is unavailable")
	}
	next, err := buildServices(config)
	if err != nil {
		return nil, err
	}
	return func() { n.active.Store(next) }, nil
}

// InitService atomically replaces the active service set. It is the boot-time
// equivalent of Prepare+commit for callers that have no persistence step.
func (n *Notifier) InitService(config *configs.NotifyConfig) error {
	commit, err := n.Prepare(config)
	if err != nil {
		return err
	}
	commit()
	return nil
}

func (n *Notifier) Send(eventType, op, message string) {
	if n == nil {
		return
	}
	active := n.active.Load()
	if active == nil || active.Disabled {
		return
	}
	title := fmt.Sprintf("[%s] %s", eventType, op)
	err := active.Send(context.Background(), title, message)
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
