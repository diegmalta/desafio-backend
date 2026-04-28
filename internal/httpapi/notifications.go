package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/integrations"
	"desafio-backend/internal/repo"
)

const (
	defaultNotifLimit = 20
	maxNotifLimit     = 100
)

func handleListNotifications(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset, ok := parsePagination(c)
		if !ok {
			return
		}
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusOK, gin.H{
				"items":  []any{},
				"limit":  limit,
				"offset": offset,
				"total":  0,
			})
			return
		}
		rows, total, err := repo.ListNotificationsByCitizen(c.Request.Context(), pool, *cid, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		items := make([]notificationJSON, 0, len(rows))
		for _, r := range rows {
			items = append(items, rowToJSON(r))
		}
		c.JSON(http.StatusOK, gin.H{
			"items":  items,
			"limit":  limit,
			"offset": offset,
			"total":  total,
		})
	}
}

func handleUnreadCount(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusOK, gin.H{"count": 0})
			return
		}
		n, err := repo.CountUnreadNotifications(c.Request.Context(), pool, *cid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": n})
	}
}

func handleMarkRead(pool *pgxpool.Pool, chamados *integrations.ChamadosClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		idStr := c.Param("id")
		nid, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
			return
		}
		ok, chamadoID, err := repo.MarkNotificationRead(c.Request.Context(), pool, *cid, nid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		scheduleChamadosReadNotify(chamados, chamadoID, cid.String())
		c.Status(http.StatusNoContent)
	}
}

func scheduleChamadosReadNotify(ch *integrations.ChamadosClient, chamadoID, citizenRef string) {
	if ch == nil || chamadoID == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := ch.NotifyRead(ctx, chamadoID, citizenRef); err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) {
				log.Printf("chamados NotifyRead: circuit open chamado_id=%s", chamadoID)
				return
			}
			log.Printf("chamados NotifyRead: %v", chamadoID)
		}
	}()
}

func handleGetNotification(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		idStr := c.Param("id")
		nid, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
			return
		}
		row, err := repo.GetNotificationByCitizen(c.Request.Context(), pool, *cid, nid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if row == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusOK, rowToJSON(*row))
	}
}

func handleMarkAllRead(pool *pgxpool.Pool, chamados *integrations.ChamadosClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		chamadosIDs, n, err := repo.MarkAllNotificationsRead(c.Request.Context(), pool, *cid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		seen := make(map[string]struct{})
		for _, ch := range chamadosIDs {
			if ch == "" {
				continue
			}
			if _, ok := seen[ch]; ok {
				continue
			}
			seen[ch] = struct{}{}
			scheduleChamadosReadNotify(chamados, ch, cid.String())
		}
		c.JSON(http.StatusOK, gin.H{"updated": n})
	}
}

type notificationJSON struct {
	ID              string    `json:"id"`
	ChamadoID       string    `json:"chamado_id"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	ReadAt          *timeJSON `json:"read_at,omitempty"`
	CreatedAt       timeJSON  `json:"created_at"`
	StatusAnterior  *string   `json:"status_anterior,omitempty"`
	StatusNovo      *string   `json:"status_novo,omitempty"`
	EventType       *string   `json:"event_type,omitempty"`
	SourceTimestamp *timeJSON `json:"source_timestamp,omitempty"`
}

type timeJSON struct{ time.Time }

func (t timeJSON) MarshalJSON() ([]byte, error) {
	return t.Time.UTC().MarshalJSON()
}

func rowToJSON(r repo.NotificationRow) notificationJSON {
	out := notificationJSON{
		ID:             r.ID.String(),
		ChamadoID:      r.ChamadoID,
		Title:          r.Title,
		Body:           r.Body,
		CreatedAt:      timeJSON{r.CreatedAt.UTC()},
		StatusAnterior: r.StatusAnterior,
		StatusNovo:     r.StatusNovo,
		EventType:      r.EventType,
	}
	if r.ReadAt != nil {
		u := r.ReadAt.UTC()
		out.ReadAt = &timeJSON{u}
	}
	if r.SourceTimestamp != nil {
		u := r.SourceTimestamp.UTC()
		out.SourceTimestamp = &timeJSON{u}
	}
	return out
}

func parsePagination(c *gin.Context) (limit, offset int, ok bool) {
	limit = defaultNotifLimit
	offset = 0
	if ls := c.Query("limit"); ls != "" {
		v, err := strconv.Atoi(ls)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return 0, 0, false
		}
		limit = v
		if limit > maxNotifLimit {
			limit = maxNotifLimit
		}
	}
	if os := c.Query("offset"); os != "" {
		v, err := strconv.Atoi(os)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_offset"})
			return 0, 0, false
		}
		offset = v
	}
	return limit, offset, true
}
