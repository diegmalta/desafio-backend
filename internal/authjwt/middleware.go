package authjwt

import (
	"context"
	"errors"
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

// ErrUnauthorized signals invalid or expired JWT, invalid CPF claim, or wrong iss/aud.
var ErrUnauthorized = errors.New("unauthorized")

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

// BearerFromRequest returns a JWT from Authorization: Bearer or ?access_token= (WebSocket / tooling).
func BearerFromRequest(c *gin.Context) (raw string, ok bool) {
	if t, ok := bearerToken(c.GetHeader("Authorization")); ok {
		return t, true
	}
	q := strings.TrimSpace(c.Query("access_token"))
	if q != "" {
		return q, true
	}
	return "", false
}

// ResolveCitizenFromToken validates HS256 JWT and maps preferred_username to citizen_id.
// Returns (nil, nil) when the token is valid but no citizen row exists yet.
func ResolveCitizenFromToken(ctx context.Context, pool *pgxpool.Pool, jwtSecret, cpfPepper, wantIssuer, wantAudience, raw string) (*uuid.UUID, error) {
	secret := []byte(jwtSecret)
	pepper := []byte(cpfPepper)
	var claims accessClaims
	token, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrUnauthorized
	}
	now := time.Now()
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(now) {
		return nil, ErrUnauthorized
	}
	if wantIssuer != "" && claims.Issuer != wantIssuer {
		return nil, ErrUnauthorized
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
			return nil, ErrUnauthorized
		}
	}
	cpf, err := identity.NormalizeCPF11(claims.PreferredUsername)
	if err != nil {
		return nil, ErrUnauthorized
	}
	fp := identity.CitizenFingerprint(pepper, cpf)
	return repo.LookupCitizenID(ctx, pool, fp)
}

// Middleware validates Bearer JWT (HS256), maps preferred_username to citizen fingerprint, loads citizen_id.
func Middleware(pool *pgxpool.Pool, jwtSecret, cpfPepper, wantIssuer, wantAudience string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		citizenID, err := ResolveCitizenFromToken(c.Request.Context(), pool, jwtSecret, cpfPepper, wantIssuer, wantAudience, raw)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
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
