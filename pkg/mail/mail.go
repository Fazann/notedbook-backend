// Package mail sends HTML emails over SMTP and renders the email templates.
package mail

import (
	"bytes"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/config"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// ErrNotConfigured is returned when SMTP_HOST or SMTP_FROM is not set.
var ErrNotConfigured = errors.New("mail: smtp is not configured")

// OTPData fills templates/otp.html.
type OTPData struct {
	AppName        string
	Code           string
	ExpiresMinutes int
}

// RenderOTP renders the one-time code email.
func RenderOTP(data OTPData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "otp.html", data); err != nil {
		return "", fmt.Errorf("render otp email: %w", err)
	}
	return buf.String(), nil
}

// Send sends an HTML email to one address. Port 465 uses implicit TLS; other ports upgrade with
// STARTTLS when the server offers it.
func Send(to, subject, html string) error {
	if config.SmtpHost == "" || config.SmtpFrom == "" {
		return ErrNotConfigured
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("mail: invalid recipient: %w", err)
	}

	msg := buildMessage(config.SmtpFromName, config.SmtpFrom, to, subject, html, time.Now())
	addr := net.JoinHostPort(config.SmtpHost, config.SmtpPort)
	var auth smtp.Auth
	if config.SmtpUsername != "" {
		auth = smtp.PlainAuth("", config.SmtpUsername, config.SmtpPassword, config.SmtpHost)
	}

	if config.SmtpPort != "465" {
		return smtp.SendMail(addr, auth, config.SmtpFrom, []string{to}, msg)
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: config.SmtpHost})
	if err != nil {
		return fmt.Errorf("mail: dial: %w", err)
	}
	c, err := smtp.NewClient(conn, config.SmtpHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mail: client: %w", err)
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}
	if err := c.Mail(config.SmtpFrom); err != nil {
		return fmt.Errorf("mail: from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mail: rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close: %w", err)
	}
	return c.Quit()
}

// buildMessage returns the raw message. Header values are encoded, so they cannot inject headers.
func buildMessage(fromName, from, to, subject, html string, now time.Time) []byte {
	fromAddr := (&mail.Address{Name: fromName, Address: from}).String()
	var b strings.Builder
	b.WriteString("From: " + fromAddr + "\r\n")
	b.WriteString("To: " + (&mail.Address{Address: to}).String() + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(html, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String())
}
