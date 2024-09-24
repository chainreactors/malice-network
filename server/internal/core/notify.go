package core

//var Notifier = &notifier{
//	notify: notify.New(),
//}
//
//type notifier struct {
//	notify *notify.Notify
//}
//
//func (n *notifier) InitService(config *configs.NotifyConfig) error {
//	if !config.Enable {
//		return nil
//	}
//	if config.Telegram.Enable {
//		tg, err := telegram.New(config.Telegram.APIKey)
//		if err != nil {
//			return err
//		}
//		tg.SetParseMode(telegram.ModeMarkdown)
//		tg.AddReceivers(config.Telegram.ChatID)
//		n.notify.UseServices(tg)
//	}
//	if config.DingTalk.Enable {
//		dt := dingding.New(&dingding.Config{
//			Token:  config.DingTalk.Token,
//			Secret: config.DingTalk.Secret,
//		})
//		n.notify.UseServices(dt)
//	}
//	if config.Lark.Enable {
//		lark := lark2.NewWebhookService(config.Lark.WebHookUrl)
//		n.notify.UseServices(lark)
//	}
//	if config.ServerChan.Enable {
//		sc := http.New()
//		sc.AddReceivers(&http.Webhook{
//			URL:         config.ServerChan.URL,
//			Method:      config.ServerChan.Method,
//			Header:      config.ServerChan.Headers,
//			ContentType: config.ServerChan.ContentType,
//			BuildPayload: func(subject, message string) (payload any) {
//				return map[string]string{
//					"subject": subject,
//					"message": message,
//				}
//			},
//		})
//		n.notify.UseServices(sc)
//	}
//	return nil
//}
//
//func (n *notifier) Send(event *Event) error {
//	var title string
//	if event.Op == consts.CtrlJobStart || event.Op == consts.CtrlJobStop {
//		title = fmt.Sprintf("%s %s", event.EventType, event.Op)
//	} else {
//		title = fmt.Sprintf("%s", event.Op)
//	}
//
//	err := n.notify.Send(context.Background(), title, event.Message)
//	if err != nil {
//		return err
//	}
//	return nil
//}
