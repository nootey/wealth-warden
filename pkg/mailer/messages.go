package mailer

import (
	"fmt"
	"time"
	"wealth-warden/pkg/utils"
)

func (m *Mailer) SendRegistrationEmail(to, displayName, token string) error {
	link := m.buildLink("auth", "validate-email?token="+token)

	data := map[string]string{
		"subjectName":      displayName,
		"registrationLink": link,
		"year":             time.Now().Format("2006"),
	}

	body, err := renderTemplate("validate-registration-email.html", data)
	if err != nil {
		return err
	}
	return m.SendEmail(to, "Please validate registration email.", body)
}

func (m *Mailer) SendConfirmationEmail(to, displayName, token string) error {
	link := m.buildLink("auth", "confirm-email?token="+token)
	body, err := renderTemplate("confirm-email.html", map[string]string{
		"subjectName": displayName,
		"confirmLink": link,
		"year":        time.Now().Format("2006"),
	})
	if err != nil {
		return err
	}
	return m.SendEmail(to, "Please confirm your email address.", body)
}

func (m *Mailer) SendPasswordResetEmail(to, displayName, token string) error {
	link := m.buildLink("auth", "password-reset?token="+token)
	body, err := renderTemplate("reset-password.html", map[string]string{
		"subjectName": displayName,
		"resetLink":   link,
		"year":        time.Now().Format("2006"),
	})
	if err != nil {
		return err
	}
	return m.SendEmail(to, "A password reset has been requested.", body)
}

func (m *Mailer) buildLink(subdomain, path string) string {
	redirectUrl := utils.GenerateHttpReleaseLink(m.globalConfig)
	return fmt.Sprintf("%s%s/%s", redirectUrl, subdomain, path)
}
