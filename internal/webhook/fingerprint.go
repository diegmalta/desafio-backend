package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
)

// CitizenFingerprint is HMAC-SHA256(pepper, cpf11digits) — 32 bytes, never store raw CPF.
func CitizenFingerprint(pepper []byte, cpf11Digits string) []byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(cpf11Digits))
	return mac.Sum(nil)
}
