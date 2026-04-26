package identity

import (
	"fmt"
	"strings"
)

// NormalizeCPF11 returns the CPF as exactly 11 ASCII digits or an error.
func NormalizeCPF11(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) != 11 {
		return "", fmt.Errorf("cpf must be exactly 11 digits")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("cpf must contain only digits")
		}
	}
	return s, nil
}
