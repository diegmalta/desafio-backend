package webhook

import (
	"errors"
	"testing"
)

func TestEventPayload_Validate_ok(t *testing.T) {
	p := &EventPayload{
		ChamadoID:      "CH-1",
		Tipo:           "status_change",
		CPF:            "12345678901",
		StatusAnterior: "em_analise",
		StatusNovo:     "em_execucao",
		Titulo:         "T",
		Descricao:      "D",
		Timestamp:      "2024-11-15T14:30:00Z",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.CPF != "12345678901" {
		t.Fatalf("cpf normalized: %q", p.CPF)
	}
}

func TestEventPayload_Validate_missingField(t *testing.T) {
	p := &EventPayload{ChamadoID: "x"}
	err := p.Validate()
	if err == nil || !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("want ErrInvalidPayload, got %v", err)
	}
}

func TestEventPayload_Validate_badCPF(t *testing.T) {
	p := &EventPayload{
		ChamadoID:  "CH-1",
		Tipo:       "status_change",
		CPF:        "123",
		StatusNovo: "x",
		Titulo:     "t",
		Descricao:  "d",
		Timestamp:  "2024-11-15T14:30:00Z",
	}
	err := p.Validate()
	if err == nil || !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("want ErrInvalidPayload, got %v", err)
	}
}

func TestEventPayload_Validate_badTimestamp(t *testing.T) {
	p := &EventPayload{
		ChamadoID:  "CH-1",
		Tipo:       "status_change",
		CPF:        "12345678901",
		StatusNovo: "x",
		Titulo:     "t",
		Descricao:  "d",
		Timestamp:  "not-a-date",
	}
	err := p.Validate()
	if err == nil || !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("want ErrInvalidPayload, got %v", err)
	}
}
