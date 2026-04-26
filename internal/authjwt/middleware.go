package authjwt

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/identity"
	"desafio-backend/internal/repo"
)

const ctxCitizenID = "authCitizenID"

// CitizenID returns the authenticated citizen id from Gin context, if any.
func CitizenID(c *gin.Context) *uuid.UUID {
	v, ok := c.Get(ctxCitizenID)
	if !ok || v == nil {
		return nil
	}
	id, ok := v.(*uuid.UUID)
	if !ok || id == nil {
		return nil
	}
	return id
}

type accessClaims struct {
	jwt.RegisteredClaims
	PreferredUsername string `json:"preferred_username"`
}

// Middleware validates Bearer JWT (HS256), maps preferred_username to citizen fingerprint, loads citizen_id.
func Middleware(pool *pgxpool.Pool, jwtSecret, cpfPepper, wantIssuer, wantAudience string) gin.HandlerFunc {
	secret := []byte(jwtSecret)
	pepper := []byte(cpfPepper)
	return func(c *gin.Context) {
		raw, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		var claims accessClaims
		token, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		now := time.Now()
		if claims.ExpiresAt == nil || !claims.ExpiresAt.After(now) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if wantIssuer != "" && claims.Issuer != wantIssuer {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		if wantAudience != "" {
			okAud := false
			for _, a := range claims.Audience {
				if a == wantAudience {
					okAud = true
					break
				}
			}
			if !okAud {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
		}
		cpf, err := identity.NormalizeCPF11(claims.PreferredUsername)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		fp := identity.CitizenFingerprint(pepper, cpf)
		ctx := c.Request.Context()
		citizenID, err := repo.LookupCitizenID(ctx, pool, fp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			c.Abort()
			return
		}
		c.Set(ctxCitizenID, citizenID)
		c.Next()
	}
}

func bearerToken(h string) (string, bool) {
	if h == "" {
		return "", false
	}
	const p = "Bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return "", false
	}
	t := strings.TrimSpace(h[len(p):])
	if t == "" {
		return "", false
	}
	return t, true
}
