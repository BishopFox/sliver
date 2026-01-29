package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/syslog"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/nikoksr/notify"
	notifyamazonses "github.com/nikoksr/notify/service/amazonses"
	notifyamazonsns "github.com/nikoksr/notify/service/amazonsns"
	notifybark "github.com/nikoksr/notify/service/bark"
	notifydingding "github.com/nikoksr/notify/service/dingding"
	notifydiscord "github.com/nikoksr/notify/service/discord"
	notifyfcm "github.com/nikoksr/notify/service/fcm"
	notifygooglechat "github.com/nikoksr/notify/service/googlechat"
	notifyhttp "github.com/nikoksr/notify/service/http"
	notifylark "github.com/nikoksr/notify/service/lark"
	notifyline "github.com/nikoksr/notify/service/line"
	notifymail "github.com/nikoksr/notify/service/mail"
	notifymailgun "github.com/nikoksr/notify/service/mailgun"
	notifymatrix "github.com/nikoksr/notify/service/matrix"
	notifymattermost "github.com/nikoksr/notify/service/mattermost"
	notifymsteams "github.com/nikoksr/notify/service/msteams"
	notifypagerduty "github.com/nikoksr/notify/service/pagerduty"
	notifyplivo "github.com/nikoksr/notify/service/plivo"
	notifypushbullet "github.com/nikoksr/notify/service/pushbullet"
	notifypushover "github.com/nikoksr/notify/service/pushover"
	notifyreddit "github.com/nikoksr/notify/service/reddit"
	notifyrocketchat "github.com/nikoksr/notify/service/rocketchat"
	notifysendgrid "github.com/nikoksr/notify/service/sendgrid"
	notifyslack "github.com/nikoksr/notify/service/slack"
	notifysyslog "github.com/nikoksr/notify/service/syslog"
	notifytelegram "github.com/nikoksr/notify/service/telegram"
	notifytextmagic "github.com/nikoksr/notify/service/textmagic"
	notifytwilio "github.com/nikoksr/notify/service/twilio"
	notifytwitter "github.com/nikoksr/notify/service/twitter"
	notifyviber "github.com/nikoksr/notify/service/viber"
	notifywebpush "github.com/nikoksr/notify/service/webpush"
	notifywechat "github.com/nikoksr/notify/service/wechat"
	notifywhatsapp "github.com/nikoksr/notify/service/whatsapp"
	"google.golang.org/api/option"
	"maunium.net/go/mautrix/id"
)

var (
	startOnce     sync.Once
	activeManager *Manager
)

type notifierEntry struct {
	name     string
	notifier notify.Notifier
	events   map[string]struct{}
}

func (e notifierEntry) allows(eventType string) bool {
	if len(e.events) == 0 {
		return true
	}
	_, ok := e.events[eventType]
	return ok
}

func Start() {
	startOnce.Do(func() {
		templateDir, err := ensureTemplatesDir()
		if err != nil {
			notificationsLog.Warnf("Failed to ensure notifications templates directory: %v", err)
		} else {
			notificationsLog.Debugf("Notifications templates directory: %s", templateDir)
		}
		serverConfig := configs.GetServerConfig()
		if serverConfig.Notifications == nil || !serverConfig.Notifications.Enabled {
			notificationsLog.Infof("Notifications are disabled")
			return
		}
		manager, err := NewManager(serverConfig.Notifications)
		if err != nil {
			notificationsLog.Warnf("Failed to initialize notifications: %v", err)
			return
		}
		if len(manager.entries) == 0 {
			notificationsLog.Warnf("Notifications enabled but no services are configured")
		}
		activeManager = manager
		activeManager.Start()
	})
}

func Stop() {
	if activeManager != nil {
		activeManager.Stop()
	}
}

func NewManager(cfg *configs.NotificationsConfig) (*Manager, error) {
	if cfg == nil || !cfg.Enabled {
		notificationsLog.Infof("Notifications disabled via config")
		return &Manager{enabled: false}, nil
	}

	expandEnv(cfg)

	entries, err := buildEntries(cfg)
	if err != nil {
		return nil, err
	}

	templateDir, err := ensureTemplatesDir()
	if err != nil {
		notificationsLog.Warnf("Failed to create notifications templates directory: %v", err)
		templateDir = ""
	}
	var renderer *templateRenderer
	if templateDir != "" {
		renderer = newTemplateRenderer(templateDir)
	}
	templates := buildTemplateSpecs(cfg)
	if len(templates) > 0 && renderer == nil {
		notificationsLog.Warnf("Templates configured but template directory is unavailable")
	}
	validateTemplateSpecs(renderer, templates)

	notificationsLog.Infof("Notifications configured with %d service(s)", len(entries))
	return &Manager{
		enabled:   cfg.Enabled,
		entries:   entries,
		renderer:  renderer,
		templates: templates,
	}, nil
}

func buildEntries(cfg *configs.NotificationsConfig) ([]notifierEntry, error) {
	if cfg == nil || cfg.Services == nil {
		return nil, nil
	}

	entries := make([]notifierEntry, 0)
	addEntry := func(name string, notifier notify.Notifier, events []string) {
		if notifier == nil {
			return
		}
		entries = append(entries, notifierEntry{
			name:     name,
			notifier: notifier,
			events:   resolveEvents(cfg.Events, events),
		})
		if len(events) > 0 {
			notificationsLog.Infof("Notifications enabled for %s with %d event filter(s)", name, len(events))
		} else if len(cfg.Events) > 0 {
			notificationsLog.Infof("Notifications enabled for %s with %d global event filter(s)", name, len(cfg.Events))
		} else {
			notificationsLog.Infof("Notifications enabled for %s (all events)", name)
		}
	}

	services := cfg.Services
	if svc := services.AmazonSES; svc != nil && svc.Enabled {
		notifier, err := buildAmazonSES(svc)
		if err != nil {
			notificationsLog.Warnf("Amazon SES notifications disabled: %v", err)
		} else {
			addEntry("amazon_ses", notifier, svc.Events)
		}
	}
	if svc := services.AmazonSNS; svc != nil && svc.Enabled {
		notifier, err := buildAmazonSNS(svc)
		if err != nil {
			notificationsLog.Warnf("Amazon SNS notifications disabled: %v", err)
		} else {
			addEntry("amazon_sns", notifier, svc.Events)
		}
	}
	if svc := services.Bark; svc != nil && svc.Enabled {
		notifier, err := buildBark(svc)
		if err != nil {
			notificationsLog.Warnf("Bark notifications disabled: %v", err)
		} else {
			addEntry("bark", notifier, svc.Events)
		}
	}
	if svc := services.DingDing; svc != nil && svc.Enabled {
		notifier, err := buildDingDing(svc)
		if err != nil {
			notificationsLog.Warnf("DingDing notifications disabled: %v", err)
		} else {
			addEntry("dingding", notifier, svc.Events)
		}
	}
	if svc := services.Discord; svc != nil && svc.Enabled {
		notifier, err := buildDiscord(svc)
		if err != nil {
			notificationsLog.Warnf("Discord notifications disabled: %v", err)
		} else {
			addEntry("discord", notifier, svc.Events)
		}
	}
	if svc := services.FCM; svc != nil && svc.Enabled {
		notifier, err := buildFCM(svc)
		if err != nil {
			notificationsLog.Warnf("FCM notifications disabled: %v", err)
		} else {
			addEntry("fcm", notifier, svc.Events)
		}
	}
	if svc := services.GoogleChat; svc != nil && svc.Enabled {
		notifier, err := buildGoogleChat(svc)
		if err != nil {
			notificationsLog.Warnf("Google Chat notifications disabled: %v", err)
		} else {
			addEntry("google_chat", notifier, svc.Events)
		}
	}
	if svc := services.HTTP; svc != nil && svc.Enabled {
		notifier, err := buildHTTP(svc)
		if err != nil {
			notificationsLog.Warnf("HTTP notifications disabled: %v", err)
		} else {
			addEntry("http", notifier, svc.Events)
		}
	}
	if svc := services.Lark; svc != nil && svc.Enabled {
		notifiers, err := buildLark(svc)
		if err != nil {
			notificationsLog.Warnf("Lark notifications disabled: %v", err)
		} else {
			for _, notifier := range notifiers {
				addEntry("lark", notifier, svc.Events)
			}
		}
	}
	if svc := services.Line; svc != nil && svc.Enabled {
		notifier, err := buildLine(svc)
		if err != nil {
			notificationsLog.Warnf("Line notifications disabled: %v", err)
		} else {
			addEntry("line", notifier, svc.Events)
		}
	}
	if svc := services.LineNotify; svc != nil && svc.Enabled {
		notifier, err := buildLineNotify(svc)
		if err != nil {
			notificationsLog.Warnf("Line Notify notifications disabled: %v", err)
		} else {
			addEntry("line_notify", notifier, svc.Events)
		}
	}
	if svc := services.Mail; svc != nil && svc.Enabled {
		notifier, err := buildMail(svc)
		if err != nil {
			notificationsLog.Warnf("Mail notifications disabled: %v", err)
		} else {
			addEntry("mail", notifier, svc.Events)
		}
	}
	if svc := services.Mailgun; svc != nil && svc.Enabled {
		notifier, err := buildMailgun(svc)
		if err != nil {
			notificationsLog.Warnf("Mailgun notifications disabled: %v", err)
		} else {
			addEntry("mailgun", notifier, svc.Events)
		}
	}
	if svc := services.Matrix; svc != nil && svc.Enabled {
		notifier, err := buildMatrix(svc)
		if err != nil {
			notificationsLog.Warnf("Matrix notifications disabled: %v", err)
		} else {
			addEntry("matrix", notifier, svc.Events)
		}
	}
	if svc := services.Mattermost; svc != nil && svc.Enabled {
		notifier, err := buildMattermost(svc)
		if err != nil {
			notificationsLog.Warnf("Mattermost notifications disabled: %v", err)
		} else {
			addEntry("mattermost", notifier, svc.Events)
		}
	}
	if svc := services.MSTeams; svc != nil && svc.Enabled {
		notifier, err := buildMSTeams(svc)
		if err != nil {
			notificationsLog.Warnf("MS Teams notifications disabled: %v", err)
		} else {
			addEntry("msteams", notifier, svc.Events)
		}
	}
	if svc := services.PagerDuty; svc != nil && svc.Enabled {
		notifier, err := buildPagerDuty(svc)
		if err != nil {
			notificationsLog.Warnf("PagerDuty notifications disabled: %v", err)
		} else {
			addEntry("pagerduty", notifier, svc.Events)
		}
	}
	if svc := services.Plivo; svc != nil && svc.Enabled {
		notifier, err := buildPlivo(svc)
		if err != nil {
			notificationsLog.Warnf("Plivo notifications disabled: %v", err)
		} else {
			addEntry("plivo", notifier, svc.Events)
		}
	}
	if svc := services.Pushbullet; svc != nil && svc.Enabled {
		notifier, err := buildPushbullet(svc)
		if err != nil {
			notificationsLog.Warnf("Pushbullet notifications disabled: %v", err)
		} else {
			addEntry("pushbullet", notifier, svc.Events)
		}
	}
	if svc := services.PushbulletSMS; svc != nil && svc.Enabled {
		notifier, err := buildPushbulletSMS(svc)
		if err != nil {
			notificationsLog.Warnf("Pushbullet SMS notifications disabled: %v", err)
		} else {
			addEntry("pushbullet_sms", notifier, svc.Events)
		}
	}
	if svc := services.Pushover; svc != nil && svc.Enabled {
		notifier, err := buildPushover(svc)
		if err != nil {
			notificationsLog.Warnf("Pushover notifications disabled: %v", err)
		} else {
			addEntry("pushover", notifier, svc.Events)
		}
	}
	if svc := services.Reddit; svc != nil && svc.Enabled {
		notifier, err := buildReddit(svc)
		if err != nil {
			notificationsLog.Warnf("Reddit notifications disabled: %v", err)
		} else {
			addEntry("reddit", notifier, svc.Events)
		}
	}
	if svc := services.RocketChat; svc != nil && svc.Enabled {
		notifier, err := buildRocketChat(svc)
		if err != nil {
			notificationsLog.Warnf("Rocket.Chat notifications disabled: %v", err)
		} else {
			addEntry("rocketchat", notifier, svc.Events)
		}
	}
	if svc := services.SendGrid; svc != nil && svc.Enabled {
		notifier, err := buildSendGrid(svc)
		if err != nil {
			notificationsLog.Warnf("SendGrid notifications disabled: %v", err)
		} else {
			addEntry("sendgrid", notifier, svc.Events)
		}
	}
	if svc := services.Slack; svc != nil && svc.Enabled {
		notifier, err := buildSlack(svc)
		if err != nil {
			notificationsLog.Warnf("Slack notifications disabled: %v", err)
		} else {
			addEntry("slack", notifier, svc.Events)
		}
	}
	if svc := services.Syslog; svc != nil && svc.Enabled {
		notifier, err := buildSyslog(svc)
		if err != nil {
			notificationsLog.Warnf("Syslog notifications disabled: %v", err)
		} else {
			addEntry("syslog", notifier, svc.Events)
		}
	}
	if svc := services.Telegram; svc != nil && svc.Enabled {
		notifier, err := buildTelegram(svc)
		if err != nil {
			notificationsLog.Warnf("Telegram notifications disabled: %v", err)
		} else {
			addEntry("telegram", notifier, svc.Events)
		}
	}
	if svc := services.TextMagic; svc != nil && svc.Enabled {
		notifier, err := buildTextMagic(svc)
		if err != nil {
			notificationsLog.Warnf("TextMagic notifications disabled: %v", err)
		} else {
			addEntry("textmagic", notifier, svc.Events)
		}
	}
	if svc := services.Twilio; svc != nil && svc.Enabled {
		notifier, err := buildTwilio(svc)
		if err != nil {
			notificationsLog.Warnf("Twilio notifications disabled: %v", err)
		} else {
			addEntry("twilio", notifier, svc.Events)
		}
	}
	if svc := services.Twitter; svc != nil && svc.Enabled {
		notifier, err := buildTwitter(svc)
		if err != nil {
			notificationsLog.Warnf("Twitter notifications disabled: %v", err)
		} else {
			addEntry("twitter", notifier, svc.Events)
		}
	}
	if svc := services.Viber; svc != nil && svc.Enabled {
		notifier, err := buildViber(svc)
		if err != nil {
			notificationsLog.Warnf("Viber notifications disabled: %v", err)
		} else {
			addEntry("viber", notifier, svc.Events)
		}
	}
	if svc := services.WebPush; svc != nil && svc.Enabled {
		notifier, err := buildWebPush(svc)
		if err != nil {
			notificationsLog.Warnf("Webpush notifications disabled: %v", err)
		} else {
			addEntry("webpush", notifier, svc.Events)
		}
	}
	if svc := services.WeChat; svc != nil && svc.Enabled {
		notifier, err := buildWeChat(svc)
		if err != nil {
			notificationsLog.Warnf("WeChat notifications disabled: %v", err)
		} else {
			addEntry("wechat", notifier, svc.Events)
		}
	}
	if svc := services.WhatsApp; svc != nil && svc.Enabled {
		notifier, err := buildWhatsApp(svc)
		if err != nil {
			notificationsLog.Warnf("WhatsApp notifications disabled: %v", err)
		} else {
			addEntry("whatsapp", notifier, svc.Events)
		}
	}

	return entries, nil
}

func buildTemplateSpecs(cfg *configs.NotificationsConfig) map[string]templateSpec {
	specs := map[string]templateSpec{}
	if cfg == nil || len(cfg.Templates) == 0 {
		return specs
	}
	for eventType, tmpl := range cfg.Templates {
		if strings.TrimSpace(eventType) == "" {
			notificationsLog.Warnf("Ignoring template with empty event type")
			continue
		}
		if tmpl == nil {
			notificationsLog.Warnf("Ignoring template for event %q: config is nil", eventType)
			continue
		}
		name := strings.TrimSpace(tmpl.Template)
		if name == "" {
			notificationsLog.Warnf("Ignoring template for event %q: template name is empty", eventType)
			continue
		}
		typ, ok := parseTemplateType(tmpl.Type)
		if !ok {
			notificationsLog.Warnf("Unknown template type %q for event %q, defaulting to text", tmpl.Type, eventType)
			typ = templateTypeText
		}
		specs[eventType] = templateSpec{name: name, typ: typ}
		notificationsLog.Infof("Configured template %q (%s) for event %q", name, typ, eventType)
	}
	return specs
}

func validateTemplateSpecs(renderer *templateRenderer, specs map[string]templateSpec) {
	if len(specs) == 0 {
		return
	}
	if renderer == nil {
		notificationsLog.Warnf("Templates configured but renderer is not available")
		return
	}
	for eventType, spec := range specs {
		path, err := resolveTemplatePath(renderer.baseDir, spec.name)
		if err != nil {
			notificationsLog.Warnf("Template %q for event %q is invalid: %v", spec.name, eventType, err)
			continue
		}
		if _, err := os.Stat(path); err != nil {
			notificationsLog.Warnf("Template %q for event %q not found at %s: %v", spec.name, eventType, path, err)
		}
	}
}

func resolveEvents(globalEvents, serviceEvents []string) map[string]struct{} {
	events := serviceEvents
	if len(events) == 0 {
		events = globalEvents
	}
	if len(events) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		set[event] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func buildAmazonSES(cfg *configs.AmazonSESConfig) (notify.Notifier, error) {
	if cfg.AccessKeyID == "" || cfg.SecretKey == "" || cfg.Region == "" || cfg.SenderAddress == "" {
		return nil, errors.New("missing amazon ses credentials")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing amazon ses receivers")
	}
	service, err := notifyamazonses.New(cfg.AccessKeyID, cfg.SecretKey, cfg.Region, cfg.SenderAddress)
	if err != nil {
		return nil, err
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing amazon ses receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildAmazonSNS(cfg *configs.AmazonSNSConfig) (notify.Notifier, error) {
	if cfg.AccessKeyID == "" || cfg.SecretKey == "" || cfg.Region == "" {
		return nil, errors.New("missing amazon sns credentials")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing amazon sns receivers")
	}
	service, err := notifyamazonsns.New(cfg.AccessKeyID, cfg.SecretKey, cfg.Region)
	if err != nil {
		return nil, err
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing amazon sns receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildBark(cfg *configs.BarkConfig) (notify.Notifier, error) {
	if cfg.DeviceKey == "" {
		return nil, errors.New("missing bark device_key")
	}
	service := notifybark.NewWithServers(cfg.DeviceKey, compactStrings(cfg.Servers)...)
	return service, nil
}

func buildDingDing(cfg *configs.DingDingConfig) (notify.Notifier, error) {
	if cfg.Token == "" || cfg.Secret == "" {
		return nil, errors.New("missing dingding token/secret")
	}
	return notifydingding.New(&notifydingding.Config{
		Token:  cfg.Token,
		Secret: cfg.Secret,
	}), nil
}

func buildDiscord(cfg *configs.DiscordConfig) (notify.Notifier, error) {
	if cfg.Token == "" {
		return nil, errors.New("missing discord token")
	}
	if len(cfg.Channels) == 0 {
		return nil, errors.New("missing discord channels")
	}
	service := notifydiscord.New()
	tokenType := strings.ToLower(strings.TrimSpace(cfg.TokenType))
	switch tokenType {
	case "", "bot":
		if err := service.AuthenticateWithBotToken(cfg.Token); err != nil {
			return nil, err
		}
	case "oauth", "oauth2":
		if err := service.AuthenticateWithOAuth2Token(cfg.Token); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported discord token type %q", cfg.TokenType)
	}
	channels := compactStrings(cfg.Channels)
	if len(channels) == 0 {
		return nil, errors.New("missing discord channels")
	}
	service.AddReceivers(channels...)
	return service, nil
}

func buildFCM(cfg *configs.FCMConfig) (notify.Notifier, error) {
	if len(cfg.DeviceTokens) == 0 {
		return nil, errors.New("missing fcm device_tokens")
	}
	ctx := context.Background()
	opts := make([]notifyfcm.Option, 0)
	if cfg.CredentialsFile != "" {
		opts = append(opts, notifyfcm.WithCredentialsFile(cfg.CredentialsFile))
	}
	if cfg.ProjectID != "" {
		opts = append(opts, notifyfcm.WithProjectID(cfg.ProjectID))
	}
	service, err := notifyfcm.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	tokens := compactStrings(cfg.DeviceTokens)
	if len(tokens) == 0 {
		return nil, errors.New("missing fcm device_tokens")
	}
	service.AddReceivers(tokens...)
	return service, nil
}

func buildGoogleChat(cfg *configs.GoogleChatConfig) (notify.Notifier, error) {
	if len(cfg.Spaces) == 0 {
		return nil, errors.New("missing google chat spaces")
	}
	options := make([]option.ClientOption, 0)
	if cfg.CredentialsFile != "" {
		options = append(options, option.WithCredentialsFile(cfg.CredentialsFile))
	}
	if cfg.CredentialsJSON != "" {
		options = append(options, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	}
	service, err := notifygooglechat.New(options...)
	if err != nil {
		return nil, err
	}
	spaces := compactStrings(cfg.Spaces)
	if len(spaces) == 0 {
		return nil, errors.New("missing google chat spaces")
	}
	service.AddReceivers(spaces...)
	return service, nil
}

func buildHTTP(cfg *configs.HTTPConfig) (notify.Notifier, error) {
	if len(cfg.URLs) == 0 && len(cfg.Webhooks) == 0 {
		return nil, errors.New("missing http urls or webhooks")
	}
	service := notifyhttp.New()
	if len(cfg.URLs) > 0 {
		urls := compactStrings(cfg.URLs)
		if len(urls) == 0 {
			notificationsLog.Warnf("HTTP notifications configured without valid urls")
		} else {
			service.AddReceiversURLs(urls...)
		}
	}
	for _, hook := range cfg.Webhooks {
		if hook.URL == "" {
			notificationsLog.Warnf("HTTP webhook configured without URL")
			continue
		}
		headers := http.Header{}
		for key, value := range hook.Headers {
			if key == "" {
				continue
			}
			headers.Add(textproto.CanonicalMIMEHeaderKey(key), value)
		}
		method := strings.ToUpper(strings.TrimSpace(hook.Method))
		if method == "" {
			method = http.MethodPost
		}
		contentType := strings.TrimSpace(hook.ContentType)
		if contentType == "" {
			contentType = "application/json; charset=utf-8"
		}
		service.AddReceivers(&notifyhttp.Webhook{
			URL:         hook.URL,
			Method:      method,
			ContentType: contentType,
			Header:      headers,
			BuildPayload: func(subject, message string) any {
				return map[string]string{
					"subject": subject,
					"message": message,
				}
			},
		})
	}
	return service, nil
}

func buildLark(cfg *configs.LarkConfig) ([]notify.Notifier, error) {
	var services []notify.Notifier
	if cfg.CustomApp != nil && cfg.CustomApp.AppID != "" && cfg.CustomApp.AppSecret != "" {
		service := notifylark.NewCustomAppService(cfg.CustomApp.AppID, cfg.CustomApp.AppSecret)
		ids, err := larkReceivers(cfg.CustomApp.Receivers)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			notificationsLog.Warnf("Lark custom app configured without receivers")
		} else {
			service.AddReceivers(ids...)
			services = append(services, service)
		}
	}
	if cfg.Webhook != nil && cfg.Webhook.URL != "" {
		services = append(services, notifylark.NewWebhookService(cfg.Webhook.URL))
	}
	if len(services) == 0 {
		return nil, errors.New("missing lark configuration")
	}
	return services, nil
}

func larkReceivers(receivers []configs.LarkReceiverConfig) ([]*notifylark.ReceiverID, error) {
	ids := make([]*notifylark.ReceiverID, 0, len(receivers))
	for _, receiver := range receivers {
		receiverType := normalizeLarkType(receiver.Type)
		switch receiverType {
		case "open_id":
			ids = append(ids, notifylark.OpenID(receiver.ID))
		case "user_id":
			ids = append(ids, notifylark.UserID(receiver.ID))
		case "union_id":
			ids = append(ids, notifylark.UnionID(receiver.ID))
		case "email":
			ids = append(ids, notifylark.Email(receiver.ID))
		case "chat_id":
			ids = append(ids, notifylark.ChatID(receiver.ID))
		case "":
			continue
		default:
			return nil, fmt.Errorf("unsupported lark receiver type %q", receiver.Type)
		}
	}
	return ids, nil
}

func normalizeLarkType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func buildLine(cfg *configs.LineConfig) (notify.Notifier, error) {
	if cfg.ChannelSecret == "" || cfg.ChannelAccessToken == "" {
		return nil, errors.New("missing line channel credentials")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing line receivers")
	}
	service, err := notifyline.New(cfg.ChannelSecret, cfg.ChannelAccessToken)
	if err != nil {
		return nil, err
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing line receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildLineNotify(cfg *configs.LineNotifyConfig) (notify.Notifier, error) {
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing line notify tokens")
	}
	service := notifyline.NewNotify()
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing line notify tokens")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildMail(cfg *configs.MailConfig) (notify.Notifier, error) {
	if cfg.SenderAddress == "" || cfg.SMTPHost == "" {
		return nil, errors.New("missing mail sender_address or smtp_host")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing mail receivers")
	}
	service := notifymail.New(cfg.SenderAddress, cfg.SMTPHost)
	if cfg.SMTPUsername != "" || cfg.SMTPPassword != "" || cfg.SMTPAuthHost != "" || cfg.SMTPIdentity != "" {
		service.AuthenticateSMTP(cfg.SMTPIdentity, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPAuthHost)
	}
	if strings.EqualFold(cfg.BodyType, "plain") || strings.EqualFold(cfg.BodyType, "text") || strings.EqualFold(cfg.BodyType, "plaintext") {
		service.BodyFormat(notifymail.PlainText)
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing mail receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildMailgun(cfg *configs.MailgunConfig) (notify.Notifier, error) {
	if cfg.Domain == "" || cfg.APIKey == "" || cfg.SenderAddress == "" {
		return nil, errors.New("missing mailgun settings")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing mailgun receivers")
	}
	service := notifymailgun.New(cfg.Domain, cfg.APIKey, cfg.SenderAddress)
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing mailgun receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildMatrix(cfg *configs.MatrixConfig) (notify.Notifier, error) {
	if cfg.UserID == "" || cfg.RoomID == "" || cfg.HomeServer == "" || cfg.AccessToken == "" {
		return nil, errors.New("missing matrix configuration")
	}
	service, err := notifymatrix.New(id.UserID(cfg.UserID), id.RoomID(cfg.RoomID), cfg.HomeServer, cfg.AccessToken)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func buildMattermost(cfg *configs.MattermostConfig) (notify.Notifier, error) {
	if cfg.URL == "" {
		return nil, errors.New("missing mattermost url")
	}
	if len(cfg.Channels) == 0 {
		return nil, errors.New("missing mattermost channels")
	}
	service := notifymattermost.New(cfg.URL)
	if cfg.LoginID != "" || cfg.Password != "" {
		ctx, cancel := context.WithTimeout(context.Background(), notificationTimeout)
		if err := service.LoginWithCredentials(ctx, cfg.LoginID, cfg.Password); err != nil {
			cancel()
			return nil, err
		}
		cancel()
	}
	channels := compactStrings(cfg.Channels)
	if len(channels) == 0 {
		return nil, errors.New("missing mattermost channels")
	}
	service.AddReceivers(channels...)
	return service, nil
}

func buildMSTeams(cfg *configs.MSTeamsConfig) (notify.Notifier, error) {
	if len(cfg.Webhooks) == 0 {
		return nil, errors.New("missing msteams webhooks")
	}
	service := notifymsteams.New()
	if cfg.DisableWebhookValidation {
		service.DisableWebhookValidation()
	}
	if cfg.WrapText {
		service.WithWrapText(true)
	}
	if cfg.UserAgent != "" {
		service.SetUseragent(cfg.UserAgent)
	}
	webhooks := compactStrings(cfg.Webhooks)
	if len(webhooks) == 0 {
		return nil, errors.New("missing msteams webhooks")
	}
	service.AddReceivers(webhooks...)
	return service, nil
}

func buildPagerDuty(cfg *configs.PagerDutyConfig) (notify.Notifier, error) {
	if cfg.Token == "" {
		return nil, errors.New("missing pagerduty token")
	}
	if cfg.FromAddress == "" || len(cfg.Receivers) == 0 {
		return nil, errors.New("missing pagerduty from_address or receivers")
	}
	service, err := notifypagerduty.New(cfg.Token)
	if err != nil {
		return nil, err
	}
	service.Config.SetFromAddress(cfg.FromAddress)
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing pagerduty receivers")
	}
	service.Config.AddReceivers(receivers...)
	if cfg.NotificationType != "" {
		service.Config.SetNotificationType(cfg.NotificationType)
	}
	if cfg.Urgency != "" {
		service.Config.SetUrgency(cfg.Urgency)
	}
	if cfg.PriorityID != "" {
		service.Config.SetPriorityID(cfg.PriorityID)
	}
	return service, nil
}

func buildPlivo(cfg *configs.PlivoConfig) (notify.Notifier, error) {
	if cfg.AuthID == "" || cfg.AuthToken == "" || cfg.Source == "" {
		return nil, errors.New("missing plivo auth or source")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing plivo receivers")
	}
	service, err := notifyplivo.New(&notifyplivo.ClientOptions{
		AuthID:    cfg.AuthID,
		AuthToken: cfg.AuthToken,
	}, &notifyplivo.MessageOptions{
		Source:         cfg.Source,
		CallbackURL:    cfg.CallbackURL,
		CallbackMethod: cfg.CallbackMethod,
	})
	if err != nil {
		return nil, err
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing plivo receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildPushbullet(cfg *configs.PushbulletConfig) (notify.Notifier, error) {
	if cfg.APIToken == "" {
		return nil, errors.New("missing pushbullet api_token")
	}
	if len(cfg.DeviceNicknames) == 0 {
		return nil, errors.New("missing pushbullet device_nicknames")
	}
	service := notifypushbullet.New(cfg.APIToken)
	devices := compactStrings(cfg.DeviceNicknames)
	if len(devices) == 0 {
		return nil, errors.New("missing pushbullet device_nicknames")
	}
	service.AddReceivers(devices...)
	return service, nil
}

func buildPushbulletSMS(cfg *configs.PushbulletSMSConfig) (notify.Notifier, error) {
	if cfg.APIToken == "" || cfg.DeviceNickname == "" {
		return nil, errors.New("missing pushbullet sms api_token or device_nickname")
	}
	if len(cfg.PhoneNumbers) == 0 {
		return nil, errors.New("missing pushbullet sms phone_numbers")
	}
	service, err := notifypushbullet.NewSMS(cfg.APIToken, cfg.DeviceNickname)
	if err != nil {
		return nil, err
	}
	numbers := compactStrings(cfg.PhoneNumbers)
	if len(numbers) == 0 {
		return nil, errors.New("missing pushbullet sms phone_numbers")
	}
	service.AddReceivers(numbers...)
	return service, nil
}

func buildPushover(cfg *configs.PushoverConfig) (notify.Notifier, error) {
	if cfg.AppToken == "" {
		return nil, errors.New("missing pushover app_token")
	}
	if len(cfg.Recipients) == 0 {
		return nil, errors.New("missing pushover recipients")
	}
	service := notifypushover.New(cfg.AppToken)
	recipients := compactStrings(cfg.Recipients)
	if len(recipients) == 0 {
		return nil, errors.New("missing pushover recipients")
	}
	service.AddReceivers(recipients...)
	return service, nil
}

func buildReddit(cfg *configs.RedditConfig) (notify.Notifier, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("missing reddit credentials")
	}
	if len(cfg.Recipients) == 0 {
		return nil, errors.New("missing reddit recipients")
	}
	service, err := notifyreddit.New(cfg.ClientID, cfg.ClientSecret, cfg.Username, cfg.Password)
	if err != nil {
		return nil, err
	}
	recipients := compactStrings(cfg.Recipients)
	if len(recipients) == 0 {
		return nil, errors.New("missing reddit recipients")
	}
	service.AddReceivers(recipients...)
	return service, nil
}

func buildRocketChat(cfg *configs.RocketChatConfig) (notify.Notifier, error) {
	if cfg.ServerURL == "" || cfg.Scheme == "" || cfg.UserID == "" || cfg.Token == "" {
		return nil, errors.New("missing rocketchat configuration")
	}
	if len(cfg.Channels) == 0 {
		return nil, errors.New("missing rocketchat channels")
	}
	service, err := notifyrocketchat.New(cfg.ServerURL, cfg.Scheme, cfg.UserID, cfg.Token)
	if err != nil {
		return nil, err
	}
	channels := compactStrings(cfg.Channels)
	if len(channels) == 0 {
		return nil, errors.New("missing rocketchat channels")
	}
	service.AddReceivers(channels...)
	return service, nil
}

func buildSendGrid(cfg *configs.SendGridConfig) (notify.Notifier, error) {
	if cfg.APIKey == "" || cfg.SenderAddress == "" || cfg.SenderName == "" {
		return nil, errors.New("missing sendgrid api_key or sender")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing sendgrid receivers")
	}
	service := notifysendgrid.New(cfg.APIKey, cfg.SenderAddress, cfg.SenderName)
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing sendgrid receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildSlack(cfg *configs.SlackConfig) (notify.Notifier, error) {
	if cfg.APIToken == "" {
		return nil, errors.New("missing slack api_token")
	}
	if len(cfg.Channels) == 0 {
		return nil, errors.New("missing slack channels")
	}
	service := notifyslack.New(cfg.APIToken)
	channels := compactStrings(cfg.Channels)
	if len(channels) == 0 {
		return nil, errors.New("missing slack channels")
	}
	service.AddReceivers(channels...)
	return service, nil
}

func buildSyslog(cfg *configs.SyslogConfig) (notify.Notifier, error) {
	priority := parseSyslogPriority(cfg.Priority)
	network := strings.TrimSpace(cfg.Network)
	address := strings.TrimSpace(cfg.Address)
	tag := strings.TrimSpace(cfg.Tag)
	if network != "" || address != "" {
		return notifysyslog.NewFromDial(network, address, priority, tag)
	}
	return notifysyslog.New(priority, tag)
}

func parseSyslogPriority(value string) syslog.Priority {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return syslog.LOG_INFO
	}
	switch value {
	case "emerg", "emergency":
		return syslog.LOG_EMERG
	case "alert":
		return syslog.LOG_ALERT
	case "crit", "critical":
		return syslog.LOG_CRIT
	case "err", "error":
		return syslog.LOG_ERR
	case "warning", "warn":
		return syslog.LOG_WARNING
	case "notice":
		return syslog.LOG_NOTICE
	case "info":
		return syslog.LOG_INFO
	case "debug":
		return syslog.LOG_DEBUG
	default:
		if numeric, err := strconv.Atoi(value); err == nil {
			return syslog.Priority(numeric)
		}
	}
	return syslog.LOG_INFO
}

func buildTelegram(cfg *configs.TelegramConfig) (notify.Notifier, error) {
	if cfg.APIToken == "" {
		return nil, errors.New("missing telegram api_token")
	}
	if len(cfg.ChatIDs) == 0 {
		return nil, errors.New("missing telegram chat_ids")
	}
	service, err := notifytelegram.New(cfg.APIToken)
	if err != nil {
		return nil, err
	}
	chatIDs, err := parseInt64s(cfg.ChatIDs)
	if err != nil {
		return nil, err
	}
	service.AddReceivers(chatIDs...)
	if cfg.ParseMode != "" {
		service.SetParseMode(cfg.ParseMode)
	}
	return service, nil
}

func buildTextMagic(cfg *configs.TextMagicConfig) (notify.Notifier, error) {
	if cfg.Username == "" || cfg.APIKey == "" {
		return nil, errors.New("missing textmagic username or api_key")
	}
	if len(cfg.PhoneNumbers) == 0 {
		return nil, errors.New("missing textmagic phone_numbers")
	}
	service := notifytextmagic.New(cfg.Username, cfg.APIKey)
	numbers := compactStrings(cfg.PhoneNumbers)
	if len(numbers) == 0 {
		return nil, errors.New("missing textmagic phone_numbers")
	}
	service.AddReceivers(numbers...)
	return service, nil
}

func buildTwilio(cfg *configs.TwilioConfig) (notify.Notifier, error) {
	if cfg.AccountSID == "" || cfg.AuthToken == "" || cfg.FromNumber == "" {
		return nil, errors.New("missing twilio credentials")
	}
	if len(cfg.PhoneNumbers) == 0 {
		return nil, errors.New("missing twilio phone_numbers")
	}
	service, err := notifytwilio.New(cfg.AccountSID, cfg.AuthToken, cfg.FromNumber)
	if err != nil {
		return nil, err
	}
	numbers := compactStrings(cfg.PhoneNumbers)
	if len(numbers) == 0 {
		return nil, errors.New("missing twilio phone_numbers")
	}
	service.AddReceivers(numbers...)
	return service, nil
}

func buildTwitter(cfg *configs.TwitterConfig) (notify.Notifier, error) {
	if cfg.ConsumerKey == "" || cfg.ConsumerSecret == "" || cfg.AccessToken == "" || cfg.AccessTokenSecret == "" {
		return nil, errors.New("missing twitter credentials")
	}
	if len(cfg.Recipients) == 0 {
		return nil, errors.New("missing twitter recipients")
	}
	service, err := notifytwitter.New(notifytwitter.Credentials{
		ConsumerKey:       cfg.ConsumerKey,
		ConsumerSecret:    cfg.ConsumerSecret,
		AccessToken:       cfg.AccessToken,
		AccessTokenSecret: cfg.AccessTokenSecret,
	})
	if err != nil {
		return nil, err
	}
	recipients := compactStrings(cfg.Recipients)
	if len(recipients) == 0 {
		return nil, errors.New("missing twitter recipients")
	}
	service.AddReceivers(recipients...)
	return service, nil
}

func buildViber(cfg *configs.ViberConfig) (notify.Notifier, error) {
	if cfg.AppKey == "" || cfg.SenderName == "" {
		return nil, errors.New("missing viber app_key or sender_name")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing viber receivers")
	}
	service := notifyviber.New(cfg.AppKey, cfg.SenderName, cfg.SenderAvatar)
	if cfg.WebhookURL != "" {
		if err := service.SetWebhook(cfg.WebhookURL); err != nil {
			return nil, err
		}
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing viber receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildWebPush(cfg *configs.WebPushConfig) (notify.Notifier, error) {
	if cfg.VAPIDPublicKey == "" || cfg.VAPIDPrivateKey == "" {
		return nil, errors.New("missing webpush vapid keys")
	}
	if len(cfg.Subscriptions) == 0 {
		return nil, errors.New("missing webpush subscriptions")
	}
	service := notifywebpush.New(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)
	subscriptions := make([]notifywebpush.Subscription, 0, len(cfg.Subscriptions))
	for _, sub := range cfg.Subscriptions {
		if sub.Endpoint == "" {
			notificationsLog.Warnf("Webpush subscription missing endpoint")
			continue
		}
		subscriptions = append(subscriptions, notifywebpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				Auth:   sub.Keys.Auth,
				P256dh: sub.Keys.P256DH,
			},
		})
	}
	if len(subscriptions) == 0 {
		return nil, errors.New("missing webpush subscriptions")
	}
	service.AddReceivers(subscriptions...)
	return service, nil
}

func buildWeChat(cfg *configs.WeChatConfig) (notify.Notifier, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" || cfg.Token == "" {
		return nil, errors.New("missing wechat configuration")
	}
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing wechat receivers")
	}
	service := notifywechat.New(&notifywechat.Config{
		AppID:          cfg.AppID,
		AppSecret:      cfg.AppSecret,
		Token:          cfg.Token,
		EncodingAESKey: cfg.EncodingAESKey,
	})
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing wechat receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func buildWhatsApp(cfg *configs.WhatsAppConfig) (notify.Notifier, error) {
	if len(cfg.Receivers) == 0 {
		return nil, errors.New("missing whatsapp receivers")
	}
	service, err := notifywhatsapp.New()
	if err != nil {
		return nil, err
	}
	receivers := compactStrings(cfg.Receivers)
	if len(receivers) == 0 {
		return nil, errors.New("missing whatsapp receivers")
	}
	service.AddReceivers(receivers...)
	return service, nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func parseInt64s(values []string) ([]int64, error) {
	out := make([]int64, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", value)
		}
		out = append(out, parsed)
	}
	if len(out) == 0 {
		return nil, errors.New("no valid integers")
	}
	return out, nil
}
