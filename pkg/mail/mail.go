package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
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

	header := make(map[string]string)
	header["From"] = from
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"utf-8\""

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
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
