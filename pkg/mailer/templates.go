package mailer

import (
	"bytes"
	"path/filepath"
	"text/template"
)

func renderTemplate(templateName string, data interface{}) (string, error) {
	tmplPath := filepath.Join("pkg", "mailer", "templates", templateName)
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
