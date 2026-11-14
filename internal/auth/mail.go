package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

func enqueueMail(tx *gorm.DB, recipient, subject, body string, expires time.Time) error {
	encrypted, err := sealSecret(body, "mail/"+recipient)
	if err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO legui_private.mail_outbox(recipient,subject,body_cipher,expires_at) VALUES(?,?,?,?)`, recipient, subject, encrypted, expires).Error
}
func MailConfigured() bool { return os.Getenv("SMTP_HOST") != "" && os.Getenv("SMTP_FROM") != "" }

// Outbox entries remain pending when SMTP is absent; no recovery link is logged.
func StartMailWorker(ctx context.Context, database *gorm.DB) {
	if !MailConfigured() {
		log.Print("Correo transaccional pendiente: configurar SMTP_HOST y SMTP_FROM; las invitaciones permiten compartir el enlace.")
		return
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := deliverOne(ctx, database); err != nil {
					log.Printf("auth mail delivery unavailable (%T); queued for retry", err)
				}
			}
		}
	}()
}
func deliverOne(ctx context.Context, database *gorm.DB) error {
	tx := database.WithContext(ctx)
	if err := tx.Exec(`UPDATE legui_private.mail_outbox SET body_cipher=NULL WHERE (expires_at<clock_timestamp() OR NOT legui_private.mail_eligible(id)) AND body_cipher IS NOT NULL AND (lease_until IS NULL OR lease_until<clock_timestamp())`).Error; err != nil {
		return err
	}
	var item struct {
		ID         string
		Recipient  string
		Subject    string
		BodyCipher string
		LeaseToken string
	}
	result := tx.Raw(`UPDATE legui_private.mail_outbox SET lease_token=gen_random_uuid(),lease_until=clock_timestamp()+interval '1 minute',attempts=attempts+1 WHERE id=(SELECT id FROM legui_private.mail_outbox WHERE delivered_at IS NULL AND body_cipher IS NOT NULL AND expires_at>clock_timestamp() AND next_attempt_at<=clock_timestamp() AND attempts<8 AND legui_private.mail_eligible(id) AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY (campaign_id IS NOT NULL),created_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING id::text,recipient,subject,body_cipher,lease_token::text`).Scan(&item)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	var eligible bool
	if e := tx.Raw(`SELECT legui_private.mail_eligible(?)`, item.ID).Scan(&eligible).Error; e != nil {
		return e
	}
	if !eligible {
		return tx.Exec(`UPDATE legui_private.mail_outbox SET body_cipher=NULL,lease_until=NULL WHERE id=? AND lease_token=?`, item.ID, item.LeaseToken).Error
	}
	body, err := openSecret(item.BodyCipher, "mail/"+item.Recipient)
	if err != nil {
		return err
	}
	if err = sendMail(item.Recipient, item.Subject, body); err != nil {
		return tx.Exec(`UPDATE legui_private.mail_outbox SET lease_until=NULL,next_attempt_at=clock_timestamp()+interval '5 minutes' WHERE id=? AND lease_token=?`, item.ID, item.LeaseToken).Error
	}
	return tx.Exec(`UPDATE legui_private.mail_outbox SET delivered_at=clock_timestamp(),body_cipher=NULL,lease_until=NULL WHERE id=? AND lease_token=?`, item.ID, item.LeaseToken).Error
}
func sendMail(recipient, subject, body string) error {
	from, err := mail.ParseAddress(os.Getenv("SMTP_FROM"))
	if err != nil {
		return fmt.Errorf("SMTP_FROM invalid")
	}
	to, err := mail.ParseAddress(recipient)
	if err != nil || to.Address != recipient {
		return fmt.Errorf("recipient invalid")
	}
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	address := net.JoinHostPort(host, port)
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	if port == "465" {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", address, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", address, 10*time.Second)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if port != "465" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP requires TLS")
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		if err = client.Auth(smtp.PlainAuth("", username, os.Getenv("SMTP_PASSWORD"), host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(recipient); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	subject = strings.ReplaceAll(strings.ReplaceAll(subject, "\r", ""), "\n", "")
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from.String(), to.String(), mime.QEncoding.Encode("utf-8", subject), strings.ReplaceAll(body, "\n", "\r\n"))
	if err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// QueueShopperMail uses the existing encrypted transactional mail outbox.
func QueueShopperMail(tx *gorm.DB, recipient, subject, body string) error {
	return enqueueMail(tx, recipient, subject, body, time.Now().Add(time.Hour))
}

// Campaign bodies and outbox messages have separate authenticated encryption domains.
func SealCampaign(id, body string) (string, error)   { return sealSecret(body, "campaign/"+id) }
func OpenCampaign(id, cipher string) (string, error) { return openSecret(cipher, "campaign/"+id) }
func QueueCommerceMail(tx *gorm.DB, recipient, subject, body, campaign, shopper, brand, event string) (string, error) {
	cipher, e := sealSecret(body, "mail/"+recipient)
	if e != nil {
		return "", e
	}
	id := uuid.NewString()
	e = tx.Exec(`INSERT INTO legui_private.mail_outbox(id,recipient,subject,body_cipher,expires_at,campaign_id,shopper_id,brand_id,order_event_id) VALUES(?,?,?,?,clock_timestamp()+interval '1 day',nullif(?,'')::uuid,nullif(?,'')::uuid,nullif(?,'')::uuid,nullif(?,'')::uuid)`, id, recipient, subject, cipher, campaign, shopper, brand, event).Error
	return id, e
}
