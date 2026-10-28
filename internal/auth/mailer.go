package auth

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"time"
)

func (a *App) sendEmail(to, subject, body string) error {
	host, port, from := os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM")
	if host == "" || port == "" || from == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_FROM are required")
	}
	if _, e := mail.ParseAddress(from); e != nil {
		return fmt.Errorf("invalid SMTP_FROM")
	}
	addr := host + ":" + port
	var auth smtp.Auth
	if user := os.Getenv("SMTP_USERNAME"); user != "" {
		auth = smtp.PlainAuth("", user, os.Getenv("SMTP_PASSWORD"), host)
	}
	msg := []byte("From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n")
	conn, e := net.DialTimeout("tcp", addr, 10*time.Second)
	if e != nil {
		return e
	}
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	client, e := smtp.NewClient(conn, host)
	if e != nil {
		_ = conn.Close()
		return e
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}
	if e = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); e != nil {
		return e
	}
	if auth != nil {
		if e = client.Auth(auth); e != nil {
			return e
		}
	}
	if e = client.Mail(from); e != nil {
		return e
	}
	if e = client.Rcpt(to); e != nil {
		return e
	}
	w, e := client.Data()
	if e != nil {
		return e
	}
	if _, e = w.Write(msg); e != nil {
		_ = w.Close()
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return client.Quit()
}
