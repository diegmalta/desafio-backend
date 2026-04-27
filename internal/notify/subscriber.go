package notify

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"desafio-backend/internal/wsbus"
)

// Subscriber forwards Redis Pub/Sub messages to the local Hub.
type Subscriber struct {
	Redis *redis.Client
	Hub   *wsbus.Hub
}

// Run subscribes until ctx is cancelled; reconnects on Redis errors.
func (s *Subscriber) Run(ctx context.Context) {
	backoff := 200 * time.Millisecond
	for {
		if ctx.Err() != nil {
			return
		}
		err := s.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("notify subscriber: %v (reconnect in %s)", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = 200 * time.Millisecond
	}
}

func (s *Subscriber) runOnce(ctx context.Context) error {
	pubsub := s.Redis.PSubscribe(ctx, citizenChannelPrefix+"*")
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if msg == nil {
				continue
			}
			if msg.Channel == "" || strings.EqualFold(msg.Channel, "subscribe") {
				continue
			}
			citizenID, err := ParseCitizenChannel(msg.Channel)
			if err != nil {
				log.Printf("notify subscriber: %v", err)
				continue
			}
			s.Hub.Dispatch(citizenID, []byte(msg.Payload))
		}
	}
}
