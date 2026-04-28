package authjwt

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/repo"
)

// TouchLastSeenMiddleware updates citizens.last_seen_at after auth (best-effort).
func TouchLastSeenMiddleware(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if id := CitizenID(c); id != nil && pool != nil {
			if err := repo.TouchCitizenLastSeen(c.Request.Context(), pool, *id); err != nil {
				log.Printf("touch last_seen: %v", err)
			}
		}
		c.Next()
	}
}
