package webhook

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type notificationOutbox struct {
	Type            string     `json:"type"`
	ID              string     `json:"id"`
	ChamadoID       string     `json:"chamado_id"`
	Title           string     `json:"title"`
	Body            string     `json:"body"`
	CreatedAt       time.Time  `json:"created_at"`
	StatusAnterior  *string    `json:"status_anterior,omitempty"`
	StatusNovo      *string    `json:"status_novo,omitempty"`
	EventType       *string    `json:"event_type,omitempty"`
	SourceTimestamp *time.Time `json:"source_timestamp,omitempty"`
}

// OutboxPayloadJSON builds the JSON stored in event_outbox and published to Redis (no CPF).
func OutboxPayloadJSON(notificationID uuid.UUID, p *EventPayload, createdAt time.Time) ([]byte, error) {
	out := notificationOutbox{
		Type:      "notification",
		ID:        notificationID.String(),
		ChamadoID: p.ChamadoID,
		Title:     p.Titulo,
		Body:      p.Descricao,
		CreatedAt: createdAt.UTC(),
	}
	if s := strings.TrimSpace(p.StatusAnterior); s != "" {
		out.StatusAnterior = &s
	}
	if s := strings.TrimSpace(p.StatusNovo); s != "" {
		out.StatusNovo = &s
	}
	if s := strings.TrimSpace(p.Tipo); s != "" {
		out.EventType = &s
	}
	if ts, err := p.ParsedTimestamp(); err == nil && ts != nil {
		u := ts.UTC()
		out.SourceTimestamp = &u
	}
	return json.Marshal(out)
}
