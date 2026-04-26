package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var errBodyTooLarge = errors.New("webhook: body too large")

func readLimitedBody(c *gin.Context, limit int64) ([]byte, error) {
	r := http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	b, err := io.ReadAll(r)
	if err != nil {
		var mb *http.MaxBytesError
		if errors.As(err, &mb) {
			return nil, errBodyTooLarge
		}
		return nil, err
	}
	return b, nil
}

func unmarshalStrictJSON(body []byte, p *EventPayload) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(p)
}

func writeWebhookError(c *gin.Context, err error) {
	if errors.Is(err, errBodyTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload_too_large"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
}
