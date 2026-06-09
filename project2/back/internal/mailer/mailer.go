package mailer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/tochka-pamyati/tochka-pamyati/internal/config"
)

type Message struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type NoopSender struct{}

func (NoopSender) Send(ctx context.Context, msg Message) error { return nil }

type SMTP struct {
	cfg config.SMTPConfig
}

func NewSMTP(cfg config.SMTPConfig) *SMTP {
	return &SMTP{cfg: cfg}
}

func (s *SMTP) Send(ctx context.Context, msg Message) error {
	if strings.TrimSpace(msg.To) == "" {
		return errors.New("missing To")
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	deadline := time.Now().Add(10 * time.Second)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	var err error

	if s.cfg.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if s.cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(extractFromEmail(s.cfg.From)); err != nil {
		return err
	}
	if err := client.Rcpt(msg.To); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	raw := buildRawEmail(s.cfg.From, msg.To, msg.Subject, msg.Body, msg.IsHTML)
	if _, err := w.Write([]byte(raw)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func extractFromEmail(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return ""
	}
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.LastIndex(from, ">"); j > i {
			return strings.TrimSpace(from[i+1 : j])
		}
	}
	return from
}

func buildRawEmail(from, to, subject, body string, isHTML bool) string {
	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	encodedFrom := encodeHeader(from)
	encodedSubject := encodeHeader(subject)

	res := fmt.Sprintf("From: %s\r\n", encodedFrom)
	res += fmt.Sprintf("To: %s\r\n", to)
	res += fmt.Sprintf("Subject: %s\r\n", encodedSubject)
	res += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	res += fmt.Sprintf("Message-ID: <%d.%s@tochkapamyati.ru>\r\n", time.Now().UnixNano(), strings.Split(extractFromEmail(from), "@")[0])
	res += "MIME-Version: 1.0\r\n"
	res += fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType)
	res += "Content-Transfer-Encoding: 8bit\r\n"
	res += "\r\n"
	res += body
	res += "\r\n"
	return res
}

func encodeHeader(input string) string {
	hasNonASCII := false
	for _, r := range input {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}
	if !hasNonASCII {
		return input
	}

	if strings.Contains(input, "<") && strings.Contains(input, ">") {
		parts := strings.SplitN(input, "<", 2)
		name := strings.TrimSpace(parts[0])
		email := strings.TrimSpace(strings.Trim(parts[1], ">"))
		return fmt.Sprintf("=?UTF-8?B?%s?= <%s>", base64.StdEncoding.EncodeToString([]byte(name)), email)
	}

	return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(input)))
}
