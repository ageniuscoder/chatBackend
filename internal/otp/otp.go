package otp

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

type Store interface { //for plugin type system for databases
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Service struct {
	DB     Store
	Digits int
	TTL    time.Duration
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

	fmt.Printf("[OTP] %s for %s: %s\n", purpose, phone, code)

	return code, nil
}

func (s *Service) Verify(phone, purpose, code string) (bool, error) {
	//cleanup Expired Code
	_, _ = s.DB.Exec(
		`DELETE FROM otp_codes WHERE expires_at <= CURRENT_TIMESTAMP`,
	)

	row := s.DB.QueryRow( //count(1) provide number of rows that matches given condition
		`SELECT COUNT(1) FROM otp_codes             
		 WHERE phone_number=? AND purpose=? AND code=? 
		   AND expires_at > CURRENT_TIMESTAMP`,
		phone, purpose, code,
	)

	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}

	if n == 1 {
		//Delete OTP after successful verification
		_, err := s.DB.Exec(
			`DELETE FROM otp_codes 
			 WHERE phone_number=? AND purpose=? AND code=?`,
			phone, purpose, code,
		)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil

}
