package webhook

import "errors"

var (
	ErrMissingSignature = errors.New("webhook: missing X-Signature-256")
	ErrInvalidSignature = errors.New("webhook: invalid signature")
	ErrInvalidPayload   = errors.New("webhook: invalid payload")
)
