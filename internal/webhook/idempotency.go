package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// IdempotencyKey returns a stable key for identical JSON re-deliveries (literal field values from payload).
func IdempotencyKey(p *EventPayload) string {
	canonical := fmt.Sprintf("v1|%s|%s|%s|%s", p.ChamadoID, p.StatusNovo, p.Timestamp, p.Tipo)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}
