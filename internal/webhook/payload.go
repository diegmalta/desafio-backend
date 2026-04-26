package webhook

import (
	"fmt"
	"strings"
	"time"

	"desafio-backend/internal/identity"
)

// EventPayload mirrors the external webhook JSON (CPF is validated then discarded from persistence paths).
type EventPayload struct {
	ChamadoID      string `json:"chamado_id"`
	Tipo           string `json:"tipo"`
	CPF            string `json:"cpf"`
	StatusAnterior string `json:"status_anterior"`
	StatusNovo     string `json:"status_novo"`
	Titulo         string `json:"titulo"`
	Descricao      string `json:"descricao"`
	Timestamp      string `json:"timestamp"`
}

// Validate checks required fields and formats. CPF must be 11 digits (no formatting).
func (p *EventPayload) Validate() error {
	if strings.TrimSpace(p.ChamadoID) == "" ||
		strings.TrimSpace(p.Tipo) == "" ||
		strings.TrimSpace(p.StatusNovo) == "" ||
		strings.TrimSpace(p.Titulo) == "" ||
		strings.TrimSpace(p.Descricao) == "" ||
		strings.TrimSpace(p.Timestamp) == "" {
		return fmt.Errorf("%w: missing required fields", ErrInvalidPayload)
	}
	digits, err := identity.NormalizeCPF11(p.CPF)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	p.CPF = digits
	if _, err := time.Parse(time.RFC3339Nano, p.Timestamp); err != nil {
		if _, err2 := time.Parse(time.RFC3339, p.Timestamp); err2 != nil {
			return fmt.Errorf("%w: invalid timestamp", ErrInvalidPayload)
		}
	}
	return nil
}

// ParsedTimestamp returns the event timestamp as UTC, if valid.
func (p *EventPayload) ParsedTimestamp() (*time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, p.Timestamp); err == nil {
		u := t.UTC()
		return &u, nil
	}
	t, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		return nil, err
	}
	u := t.UTC()
	return &u, nil
}
