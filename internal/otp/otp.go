package otp

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Store is the interface for our database operations.
// It is modified to include the Begin() method for transactions.
type Store interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Begin() (*sql.Tx, error)
}

type Service struct {
	DB          Store
	Digits      int
	TTL         time.Duration
	TwilioSID   string
	TwilioToken string
	TwilioFrom  string // your Twilio phone number
}

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

func (s *Service) Genrate(phone, purpose string) (string, error) {
	code, err := randomDigit(s.Digits)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(s.TTL)

	_, err = s.DB.Exec(
		`INSERT INTO otp_codes (phone_number, code, purpose, expires_at)
         VALUES (?, ?, ?, ?)`,
		phone, code, purpose, expiresAt,
	)
	if err != nil {
		return "", err
	}

	// ✅ Send OTP via Twilio SMS //stub sender
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: s.TwilioSID,
		Password: s.TwilioToken,
	})

	params := &openapi.CreateMessageParams{}
	params.SetTo(phone)          // user's phone number
	params.SetFrom(s.TwilioFrom) // your Twilio number
	params.SetBody(fmt.Sprintf("Your verification code for %s for MmChat : %s", purpose, code))

	_, err = client.Api.CreateMessage(params)
	if err != nil {
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	return code, nil
}

func (s *Service) Verify(phone, purpose, code string) (bool, error) {
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
         WHERE phone_number=? AND purpose=? AND code=? 
           AND expires_at > CURRENT_TIMESTAMP`,
		phone, purpose, code,
	)

	if err := row.Scan(&n); err != nil {
		return false, err
	}

	if n == 1 {
		// Delete the OTP after successful verification.
		_, err := tx.Exec(
			`DELETE FROM otp_codes 
             WHERE phone_number=? AND purpose=? AND code=?`,
			phone, purpose, code,
		)
		if err != nil {
			return false, err
		}
		// Commit the transaction to save the changes.
		return true, tx.Commit()
	}

	return false, nil
}
