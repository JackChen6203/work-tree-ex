package mailer

import (
	"bytes"
	htmltmpl "html/template"
	"strings"
	texttmpl "text/template"
	"time"
)

type DigestEntry struct {
	Title     string
	Body      string
	Link      string
	CreatedAt string
}

type localeTemplates struct {
	magicSubject string
	magicText    string
	magicHTML    string

	inviteSubject string
	inviteText    string
	inviteHTML    string

	inviteReminderSubject string
	inviteReminderText    string
	inviteReminderHTML    string

	digestDailySubject  string
	digestWeeklySubject string
	digestText          string
	digestHTML          string
}

func BuildMagicLinkMessage(toEmail, locale, code, verifyURL string, expiresAt time.Time) Message {
	pack := templatesForLocale(locale)
	data := map[string]any{
		"Code":       code,
		"VerifyURL":  verifyURL,
		"ExpiresAt":  expiresAt.UTC().Format("2006-01-02 15:04 MST"),
		"IssuedAt":   time.Now().UTC().Format("2006-01-02 15:04 MST"),
		"LocaleCode": normalizeLocale(locale),
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: renderTextTemplate(pack.magicSubject, data),
		HTML:    renderHTMLTemplate(pack.magicHTML, data),
		Text:    renderTextTemplate(pack.magicText, data),
	}
}

func BuildInviteMessage(toEmail, locale, inviterName, tripName, acceptURL string, expiresAt time.Time) Message {
	pack := templatesForLocale(locale)
	data := map[string]any{
		"InviterName": inviterName,
		"TripName":    tripName,
		"AcceptURL":   acceptURL,
		"ExpiresAt":   expiresAt.UTC().Format("2006-01-02 15:04 MST"),
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: renderTextTemplate(pack.inviteSubject, data),
		HTML:    renderHTMLTemplate(pack.inviteHTML, data),
		Text:    renderTextTemplate(pack.inviteText, data),
	}
}

func BuildInviteReminderMessage(toEmail, locale, inviterName, tripName, acceptURL string, expiresAt time.Time) Message {
	pack := templatesForLocale(locale)
	data := map[string]any{
		"InviterName": inviterName,
		"TripName":    tripName,
		"AcceptURL":   acceptURL,
		"ExpiresAt":   expiresAt.UTC().Format("2006-01-02 15:04 MST"),
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: renderTextTemplate(pack.inviteReminderSubject, data),
		HTML:    renderHTMLTemplate(pack.inviteReminderHTML, data),
		Text:    renderTextTemplate(pack.inviteReminderText, data),
	}
}

func BuildTripDigestMessage(toEmail, locale, frequency string, generatedAt time.Time, entries []DigestEntry) Message {
	pack := templatesForLocale(locale)
	subjectTemplate := pack.digestDailySubject
	if strings.EqualFold(strings.TrimSpace(frequency), "weekly") {
		subjectTemplate = pack.digestWeeklySubject
	}
	data := map[string]any{
		"GeneratedAt": generatedAt.UTC().Format("2006-01-02 15:04 MST"),
		"Frequency":   strings.ToLower(strings.TrimSpace(frequency)),
		"Entries":     entries,
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: renderTextTemplate(subjectTemplate, data),
		HTML:    renderHTMLTemplate(pack.digestHTML, data),
		Text:    renderTextTemplate(pack.digestText, data),
	}
}

func templatesForLocale(locale string) localeTemplates {
	if isZhLocale(locale) {
		return localeTemplates{
			magicSubject: "你的登入驗證碼",
			magicText: `你的驗證碼是 {{.Code}}。
此驗證碼將於 {{.ExpiresAt}} 到期。

若 App 支援一鍵登入，請開啟：{{.VerifyURL}}`,
			magicHTML: `<p>你的驗證碼是 <strong>{{.Code}}</strong>。</p><p>此驗證碼將於 {{.ExpiresAt}} 到期。</p><p><a href="{{.VerifyURL}}">開啟驗證連結</a></p>`,

			inviteSubject: "你收到新的旅程邀請",
			inviteText: `{{.InviterName}} 邀請你加入行程「{{.TripName}}」。
接受邀請：{{.AcceptURL}}
邀請將於 {{.ExpiresAt}} 到期。`,
			inviteHTML: `<p>{{.InviterName}} 邀請你加入行程「<strong>{{.TripName}}</strong>」。</p><p><a href="{{.AcceptURL}}">接受邀請</a></p><p>邀請將於 {{.ExpiresAt}} 到期。</p>`,

			inviteReminderSubject: "提醒：旅程邀請即將到期",
			inviteReminderText: `提醒：{{.InviterName}} 邀請你加入行程「{{.TripName}}」。
接受邀請：{{.AcceptURL}}
邀請將於 {{.ExpiresAt}} 到期。`,
			inviteReminderHTML: `<p>提醒：{{.InviterName}} 邀請你加入行程「<strong>{{.TripName}}</strong>」。</p><p><a href="{{.AcceptURL}}">接受邀請</a></p><p>邀請將於 {{.ExpiresAt}} 到期。</p>`,

			digestDailySubject:  "每日旅程更新摘要",
			digestWeeklySubject: "每週旅程更新摘要",
			digestText: `以下是你的{{if eq .Frequency "weekly"}}每週{{else}}每日{{end}}更新（{{.GeneratedAt}}）：
{{range .Entries}}- [{{.CreatedAt}}] {{.Title}}
  {{.Body}}
  {{.Link}}
{{else}}- 本期沒有新的更新。{{end}}`,
			digestHTML: `<p>以下是你的{{if eq .Frequency "weekly"}}每週{{else}}每日{{end}}更新（{{.GeneratedAt}}）：</p><ul>{{range .Entries}}<li><strong>[{{.CreatedAt}}] {{.Title}}</strong><br/>{{.Body}}<br/><a href="{{.Link}}">{{.Link}}</a></li>{{else}}<li>本期沒有新的更新。</li>{{end}}</ul>`,
		}
	}

	return localeTemplates{
		magicSubject: "Your magic login code",
		magicText: `Your verification code is {{.Code}}.
This code expires at {{.ExpiresAt}}.

If your app supports one-click verify, open: {{.VerifyURL}}`,
		magicHTML: `<p>Your verification code is <strong>{{.Code}}</strong>.</p><p>This code expires at {{.ExpiresAt}}.</p><p><a href="{{.VerifyURL}}">Open verification link</a></p>`,

		inviteSubject: "You're invited to a trip",
		inviteText: `{{.InviterName}} invited you to join trip "{{.TripName}}".
Accept invitation: {{.AcceptURL}}
This invitation expires at {{.ExpiresAt}}.`,
		inviteHTML: `<p>{{.InviterName}} invited you to join trip <strong>{{.TripName}}</strong>.</p><p><a href="{{.AcceptURL}}">Accept invitation</a></p><p>This invitation expires at {{.ExpiresAt}}.</p>`,

		inviteReminderSubject: "Reminder: your trip invitation is expiring soon",
		inviteReminderText: `Reminder: {{.InviterName}} invited you to join trip "{{.TripName}}".
Accept invitation: {{.AcceptURL}}
This invitation expires at {{.ExpiresAt}}.`,
		inviteReminderHTML: `<p>Reminder: {{.InviterName}} invited you to join trip <strong>{{.TripName}}</strong>.</p><p><a href="{{.AcceptURL}}">Accept invitation</a></p><p>This invitation expires at {{.ExpiresAt}}.</p>`,

		digestDailySubject:  "Daily trip update digest",
		digestWeeklySubject: "Weekly trip update digest",
		digestText: `Here is your {{if eq .Frequency "weekly"}}weekly{{else}}daily{{end}} digest ({{.GeneratedAt}}):
{{range .Entries}}- [{{.CreatedAt}}] {{.Title}}
  {{.Body}}
  {{.Link}}
{{else}}- No new updates for this period.{{end}}`,
		digestHTML: `<p>Here is your {{if eq .Frequency "weekly"}}weekly{{else}}daily{{end}} digest ({{.GeneratedAt}}):</p><ul>{{range .Entries}}<li><strong>[{{.CreatedAt}}] {{.Title}}</strong><br/>{{.Body}}<br/><a href="{{.Link}}">{{.Link}}</a></li>{{else}}<li>No new updates for this period.</li>{{end}}</ul>`,
	}
}

func renderTextTemplate(templateBody string, data any) string {
	tpl, err := texttmpl.New("text-template").Parse(strings.TrimSpace(templateBody))
	if err != nil {
		return strings.TrimSpace(templateBody)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return strings.TrimSpace(templateBody)
	}
	return strings.TrimSpace(buf.String())
}

func renderHTMLTemplate(templateBody string, data any) string {
	tpl, err := htmltmpl.New("html-template").Parse(strings.TrimSpace(templateBody))
	if err != nil {
		return strings.TrimSpace(templateBody)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return strings.TrimSpace(templateBody)
	}
	return strings.TrimSpace(buf.String())
}

func isZhLocale(locale string) bool {
	return strings.HasPrefix(normalizeLocale(locale), "zh")
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}
