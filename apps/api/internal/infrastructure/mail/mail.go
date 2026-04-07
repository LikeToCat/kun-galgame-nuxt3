package mail

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"kun-galgame-api/pkg/config"
)

type Mailer struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewMailer(cfg config.MailConfig) *Mailer {
	if cfg.Host == "" {
		slog.Warn("邮件服务未配置")
		return nil
	}
	return &Mailer{
		host: cfg.Host,
		port: cfg.Port,
		user: cfg.User,
		pass: cfg.Password,
		from: cfg.From,
	}
}

func (m *Mailer) Send(to, subject, body string) error {
	auth := smtp.PlainAuth("", m.user, m.pass, m.host)
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
