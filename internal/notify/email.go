package notify

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// NotifyAdminNewRegistration sends a simple email to ADMIN_EMAIL if SMTP is configured.
func NotifyAdminNewRegistration(userEmail, role string) error {
	admin := os.Getenv("ADMIN_EMAIL")
	if admin == "" {
		return nil // nothing to do
	}
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil // email disabled
	}
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}

	addr := host
	if port != "" {
		addr = fmt.Sprintf("%s:%s", host, port)
	}

	subject := "New user registration pending approval"
	body := fmt.Sprintf("A new user has registered:\n\nEmail: %s\nRole: %s\n\nPlease review and approve in the admin panel.", userEmail, role)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", admin),
		fmt.Sprintf("Subject: %s", subject),
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, []string{admin}, []byte(msg))
}
