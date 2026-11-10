package mail

import (
	"bytes"
	"errors"
	"html/template"
	"strings"
)

type Templater struct{}

func (Templater) Render(tmpl *template.Template, payload map[string]any) (string, string, error) {
	var subject bytes.Buffer
	if err := tmpl.ExecuteTemplate(&subject, "subject", payload); err != nil {
		return "", "", err
	}

	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, "body", payload); err != nil {
		return "", "", err
	}

	subjectText := strings.TrimSpace(subject.String())
	if subjectText == "" || strings.ContainsAny(subjectText, "\r\n") {
		return "", "", errors.New("invalid subject")
	}
	if strings.TrimSpace(body.String()) == "" {
		return "", "", errors.New("empty body")
	}

	return subjectText, body.String(), nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"required": requiredValue,
	}
}

func requiredValue(v any) (string, error) {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", errors.New("missing template value")
	}
	return s, nil
}
