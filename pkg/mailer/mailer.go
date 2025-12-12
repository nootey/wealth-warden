package mailer

import (
	"wealth-warden/pkg/config"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	dialer       *gomail.Dialer
	from         string
	fromName     string
	globalConfig *config.Config
}

type MailConfig struct {
	From     string
	FromName string
}

func NewMailer(cfg *config.Config, mCfg *MailConfig) *Mailer {

	if cfg.Mailer.Host == "" {
		return nil
	}

	dialer := gomail.NewDialer(cfg.Mailer.Host, cfg.Mailer.Port, cfg.Mailer.Username, cfg.Mailer.Password)

	return &Mailer{
		dialer:       dialer,
		from:         mCfg.From,
		fromName:     mCfg.FromName,
		globalConfig: cfg,
	}
}

func (m *Mailer) SendEmail(to, subject, htmlBody string) error {
	msg := gomail.NewMessage()
	msg.SetAddressHeader("From", m.from, m.fromName)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)

	return m.dialer.DialAndSend(msg)
}
