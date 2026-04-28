package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/repo"
)

func handleCitizenMe(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		row, err := repo.GetCitizenByID(c.Request.Context(), pool, *cid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if row == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		unread, err := repo.CountUnreadNotifications(c.Request.Context(), pool, *cid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		var prefs any
		if len(row.Preferences) > 0 {
			if err := json.Unmarshal(row.Preferences, &prefs); err != nil {
				prefs = map[string]any{}
			}
		} else {
			prefs = map[string]any{}
		}
		out := gin.H{
			"citizen_id":   row.ID.String(),
			"created_at":   timeJSON{row.CreatedAt.UTC()},
			"preferences":  prefs,
			"unread_count": unread,
		}
		if row.LastSeenAt != nil {
			out["last_seen_at"] = timeJSON{row.LastSeenAt.UTC()}
		}
		c.JSON(http.StatusOK, out)
	}
}
