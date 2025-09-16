package otp

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Store is the interface for our database operations.
// It is modified to include the Begin() method for transactions.
type Store interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Begin() (*sql.Tx, error)
}

// Service holds OTP configuration and SendGrid client details.
type Service struct {
	DB             Store
	Digits         int
	TTL            time.Duration
	SendGridAPIKey string
	SendGridFrom   string // verified sender email
}

// randomDigit generates a secure random n-digit string.
func randomDigit(n int) (string, error) {
	res := make([]byte, n)
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		res[i] = byte('0' + v.Int64())
	}
	return string(res), nil
}

// Genrate creates and stores an OTP, then sends it via email using SendGrid.
func (s *Service) Genrate(email, purpose string) (string, error) {
	code, err := randomDigit(s.Digits)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(s.TTL)

	// Store OTP in DB
	_, err = s.DB.Exec(
		`INSERT INTO otp_codes (email, code, purpose, expires_at)
         VALUES (?, ?, ?, ?)`,
		email, code, purpose, expiresAt,
	)
	if err != nil {
		return "", err
	}

	// Send OTP via SendGrid
	from := mail.NewEmail("MmChat OTP Service", s.SendGridFrom)
	to := mail.NewEmail("User", email)
	subject := "Your OTP Code"
	plainText := fmt.Sprintf("Your OTP for %s in MmChat is: %s (valid for %d minutes)", purpose, code, int(s.TTL.Minutes()))
	htmlContent := fmt.Sprintf("<p>Your OTP for <b>%s</b> in MmChat is:</p><h2>%s</h2><p>Valid for %d minutes.</p>", purpose, code, int(s.TTL.Minutes()))

	message := mail.NewSingleEmail(from, subject, to, plainText, htmlContent)
	client := sendgrid.NewSendClient(s.SendGridAPIKey)
	_, err = client.Send(message)
	if err != nil {
		return "", fmt.Errorf("failed to send OTP email: %w", err)
	}

	return code, nil
}

// Verify checks if the OTP is valid and not expired.
func (s *Service) Verify(email, purpose, code string) (bool, error) {
	// Begin a new transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	// Defer a rollback in case of error. It's a no-op if commit is called.
	defer tx.Rollback()

	// Cleanup expired codes inside the transaction.
	_, _ = tx.Exec(`DELETE FROM otp_codes WHERE expires_at <= CURRENT_TIMESTAMP`)

	var n int
	row := tx.QueryRow(
		`SELECT COUNT(1) FROM otp_codes             
         WHERE email=? AND purpose=? AND code=? 
           AND expires_at > CURRENT_TIMESTAMP`,
		email, purpose, code,
	)

	if err := row.Scan(&n); err != nil {
		return false, err
	}

	if n == 1 {
		// Delete the OTP after successful verification.
		_, err := tx.Exec(
			`DELETE FROM otp_codes 
             WHERE email=? AND purpose=? AND code=?`,
			email, purpose, code,
		)
		if err != nil {
			return false, err
		}
		// Commit the transaction to save the changes.
		return true, tx.Commit()
	}

	return false, nil
}
