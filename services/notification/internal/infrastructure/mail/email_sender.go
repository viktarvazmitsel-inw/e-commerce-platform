package mail

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
)

var ErrUnableToSendEmail = errors.New("unable to send email")

//go:embed templates/*.html
var templateFS embed.FS

type smtpSendFunc func(ctx context.Context, addr, host, from string, to []string, msg []byte) error

type EmailSender struct {
	host      string
	port      string
	from      string
	templates map[string]*template.Template
	templater Templater
	send      smtpSendFunc
}

func NewEmailSender(host, port, from string) (*EmailSender, error) {
	if err := validateSMTP(host, port, from); err != nil {
		return nil, err
	}

	templates, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	return &EmailSender{
		host:      host,
		port:      port,
		from:      from,
		templates: templates,
		templater: Templater{},
		send:      sendSMTP,
	}, nil
}

func (s *EmailSender) Send(ctx context.Context, event, receiver string, payload map[string]any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrUnableToSendEmail, err)
	}

	tmpl, ok := s.templates[event]
	if !ok {
		return fmt.Errorf("%w: unknown event %q", ErrUnableToSendEmail, event)
	}

	subject, body, err := s.templater.Render(tmpl, payload)
	if err != nil {
		return fmt.Errorf("%w: render message: %w", ErrUnableToSendEmail, err)
	}

	msg := buildMessage(s.from, receiver, subject, body)
	addr := net.JoinHostPort(s.host, s.port)
	if err := s.send(ctx, addr, s.host, s.from, []string{receiver}, msg); err != nil {
		return fmt.Errorf("%w: %w", ErrUnableToSendEmail, err)
	}
	return nil
}

func loadTemplates() (map[string]*template.Template, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("read templates: %w", err)
	}

	templates := make(map[string]*template.Template, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		event := strings.TrimSuffix(entry.Name(), ".html")
		tmpl, err := template.New(entry.Name()).Funcs(templateFuncs()).ParseFS(templateFS, "templates/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", entry.Name(), err)
		}
		templates[event] = tmpl
	}
	if len(templates) == 0 {
		return nil, errors.New("no email templates")
	}
	return templates, nil
}

func buildMessage(from, to, subject, body string) []byte {
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: text/html; charset=UTF-8\r\n")
	fmt.Fprintf(&msg, "\r\n")
	msg.WriteString(body)
	return msg.Bytes()
}

func sendSMTP(ctx context.Context, addr, host, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	stopWatch := make(chan struct{})
	defer close(stopWatch)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-stopWatch:
		}
	}()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func validateSMTP(host, port, from string) error {
	if strings.TrimSpace(host) == "" || strings.ContainsAny(host, "\r\n") {
		return errors.New("smtp host is empty")
	}
	if strings.TrimSpace(port) == "" || strings.ContainsAny(port, "\r\n") {
		return errors.New("smtp port is empty")
	}
	if strings.TrimSpace(from) == "" || strings.ContainsAny(from, "\r\n") || !strings.Contains(from, "@") {
		return errors.New("smtp from is invalid")
	}
	return nil
}
