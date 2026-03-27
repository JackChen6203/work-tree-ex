package mailer

import (
	"fmt"
	"html"
	"strings"
	"time"
)

func BuildMagicLinkMessage(toEmail, locale, code, verifyURL string, expiresAt time.Time) Message {
	expireText := expiresAt.UTC().Format("2006-01-02 15:04 MST")
	subject := "Your magic login code"
	text := fmt.Sprintf(
		"Your verification code is %s.\nThis code expires at %s.\n\nIf your app supports one-click verify, open: %s",
		code,
		expireText,
		verifyURL,
	)
	htmlBody := fmt.Sprintf(
		`<p>Your verification code is <strong>%s</strong>.</p><p>This code expires at %s.</p><p><a href="%s">Open verification link</a></p>`,
		html.EscapeString(code),
		html.EscapeString(expireText),
		html.EscapeString(verifyURL),
	)

	if isZhLocale(locale) {
		subject = "你的登入驗證碼"
		text = fmt.Sprintf(
			"你的驗證碼是 %s。\n此驗證碼將於 %s 到期。\n\n若 App 支援一鍵登入，請開啟：%s",
			code,
			expireText,
			verifyURL,
		)
		htmlBody = fmt.Sprintf(
			`<p>你的驗證碼是 <strong>%s</strong>。</p><p>此驗證碼將於 %s 到期。</p><p><a href="%s">開啟驗證連結</a></p>`,
			html.EscapeString(code),
			html.EscapeString(expireText),
			html.EscapeString(verifyURL),
		)
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: subject,
		HTML:    htmlBody,
		Text:    text,
	}
}

func BuildInviteMessage(toEmail, locale, inviterName, tripName, acceptURL string, expiresAt time.Time) Message {
	subject := "You're invited to a trip"
	expireText := expiresAt.UTC().Format("2006-01-02 15:04 MST")
	text := fmt.Sprintf(
		"%s invited you to join trip \"%s\".\nAccept invitation: %s\nThis invitation expires at %s.",
		inviterName,
		tripName,
		acceptURL,
		expireText,
	)
	htmlBody := fmt.Sprintf(
		`<p>%s invited you to join trip <strong>%s</strong>.</p><p><a href="%s">Accept invitation</a></p><p>This invitation expires at %s.</p>`,
		html.EscapeString(inviterName),
		html.EscapeString(tripName),
		html.EscapeString(acceptURL),
		html.EscapeString(expireText),
	)

	if isZhLocale(locale) {
		subject = "你收到新的旅程邀請"
		text = fmt.Sprintf(
			"%s 邀請你加入行程「%s」。\n接受邀請：%s\n邀請將於 %s 到期。",
			inviterName,
			tripName,
			acceptURL,
			expireText,
		)
		htmlBody = fmt.Sprintf(
			`<p>%s 邀請你加入行程「<strong>%s</strong>」。</p><p><a href="%s">接受邀請</a></p><p>邀請將於 %s 到期。</p>`,
			html.EscapeString(inviterName),
			html.EscapeString(tripName),
			html.EscapeString(acceptURL),
			html.EscapeString(expireText),
		)
	}

	return Message{
		To:      []string{strings.TrimSpace(toEmail)},
		Subject: subject,
		HTML:    htmlBody,
		Text:    text,
	}
}

func isZhLocale(locale string) bool {
	value := strings.ToLower(strings.TrimSpace(locale))
	return strings.HasPrefix(value, "zh")
}
