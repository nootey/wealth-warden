package mailer

import (
	"bytes"
	"html/template"
	"path/filepath"
)

func renderTemplate(templateName string, data interface{}) (string, error) {
	dir := filepath.Join("storage", "mailer-templates")
	t, err := template.ParseFiles(filepath.Join(dir, "base.html"), filepath.Join(dir, templateName))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base.html", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
