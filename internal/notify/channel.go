package notify

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const citizenChannelPrefix = "notif:citizen:"

// CitizenChannel is the Redis Pub/Sub channel for one citizen.
func CitizenChannel(citizenID uuid.UUID) string {
	return citizenChannelPrefix + citizenID.String()
}

// ParseCitizenChannel extracts citizen id from a full Redis channel name.
func ParseCitizenChannel(channel string) (uuid.UUID, error) {
	if !strings.HasPrefix(channel, citizenChannelPrefix) {
		return uuid.Nil, fmt.Errorf("notify: unexpected channel %q", channel)
	}
	s := strings.TrimPrefix(channel, citizenChannelPrefix)
	return uuid.Parse(s)
}
