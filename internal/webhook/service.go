package webhook

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/repo"
)

const maxWebhookBodyBytes = 512 << 10

// Service handles webhook verification and persistence.
type Service struct {
	pool   *pgxpool.Pool
	secret []byte
	pepper []byte
}

// NewService builds a webhook service. webhookSecret and cpfPepper must be non-empty (enforced in main).
func NewService(pool *pgxpool.Pool, webhookSecret, cpfPepper string) *Service {
	return &Service{
		pool:   pool,
		secret: []byte(webhookSecret),
		pepper: []byte(cpfPepper),
	}
}

// HandlePOST is the Gin handler for POST /webhook.
func (s *Service) HandlePOST(c *gin.Context) {
	body, err := readLimitedBody(c, maxWebhookBodyBytes)
	if err != nil {
		writeWebhookError(c, err)
		return
	}
	sigHeader := c.GetHeader("X-Signature-256")
	if err := VerifySignature(body, s.secret, sigHeader); err != nil {
		if errors.Is(err, ErrMissingSignature) || errors.Is(err, ErrInvalidSignature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	var p EventPayload
	if err := unmarshalStrictJSON(body, &p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := p.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	inserted, err := s.persist(c.Request.Context(), &p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	if !inserted {
		c.JSON(http.StatusOK, gin.H{"ok": true, "duplicate": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) persist(ctx context.Context, p *EventPayload) (inserted bool, err error) {
	fp := CitizenFingerprint(s.pepper, p.CPF)
	idem := IdempotencyKey(p)
	srcTS, err := p.ParsedTimestamp()
	if err != nil {
		return false, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	params := repo.WebhookInsertParams{
		Fingerprint:     fp,
		ChamadoID:       p.ChamadoID,
		Title:           p.Titulo,
		Body:            p.Descricao,
		IdempotencyKey:  idem,
		StatusAnterior:  p.StatusAnterior,
		StatusNovo:      p.StatusNovo,
		EventType:       p.Tipo,
		SourceTimestamp: srcTS,
	}

	ok, _, err := repo.InsertNotificationIdempotent(ctx, tx, params)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return ok, nil
}
