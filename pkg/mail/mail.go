package mail

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"time"
)

// SendMail mengirim email menggunakan SMTP dengan dukungan port 465 (Implicit TLS).
func SendMail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	if from == "" {
		from = user
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	// TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}

	// Dial TLS (Port 465 biasanya menggunakan Implicit TLS)
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("gagal koneksi TLS: %v", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("gagal membuat SMTP client: %v", err)
	}
	defer c.Quit()

	// Authentication
	auth := smtp.PlainAuth("", user, pass, host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("gagal autentikasi SMTP: %v", err)
	}

	// Set the sender and recipient
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}

	// Send the email body
	w, err := c.Data()
	if err != nil {
		return err
	}

	date := time.Now().Format(time.RFC1123Z)
	randBytes := make([]byte, 12)
	rand.Read(randBytes)
	messageID := fmt.Sprintf("<%x@%s>", randBytes, host)

	message := fmt.Sprintf("Date: %s\r\n", date)
	message += fmt.Sprintf("From: Algonova Feedback <%s>\r\n", from)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += fmt.Sprintf("Message-ID: %s\r\n", messageID)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	message += "X-Mailer: Algonova-Mailer\r\n"
	message += "Precedence: bulk\r\n"
	message += "X-Auto-Response-Suppress: All\r\n"
	message += "\r\n" + body

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}
