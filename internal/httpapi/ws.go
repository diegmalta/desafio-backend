package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/wsbus"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWS(deps *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.Pool == nil || deps.Hub == nil || deps.JWTSecret == "" || deps.CPFPepper == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
			return
		}
		raw, ok := authjwt.BearerFromRequest(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		id, err := authjwt.ResolveCitizenFromToken(c.Request.Context(), deps.Pool, deps.JWTSecret, deps.CPFPepper, deps.JWTIssuer, deps.JWTAudience, raw)
		if err != nil {
			if errors.Is(err, authjwt.ErrUnauthorized) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
			return
		}
		if id == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := wsbus.NewClient(deps.Hub, *id, conn, deps.WSWriteTimeout, deps.WSPingInterval, deps.WSPongWait, deps.WSReadLimit)
		client.Run()
	}
}
