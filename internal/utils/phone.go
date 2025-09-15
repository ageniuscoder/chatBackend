package utils

import (
	"fmt"

	"github.com/nyaruka/phonenumbers"
)

// NormalizePhone converts raw phone input into E.164 format (+<countrycode><number>).
// defaultRegion is the ISO country code like "IN", "US", etc.
func NormalizePhone(raw string, defaultRegion string) (string, error) {
	num, err := phonenumbers.Parse(raw, defaultRegion)
	if err != nil {
		return "", err
	}
	if !phonenumbers.IsValidNumber(num) {
		return "", fmt.Errorf("invalid phone number")
	}
	return phonenumbers.Format(num, phonenumbers.E164), nil
}
