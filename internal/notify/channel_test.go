package notify

import (
	"testing"

	"github.com/google/uuid"
)

func TestCitizenChannel_roundTrip(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ch := CitizenChannel(id)
	got, err := ParseCitizenChannel(ch)
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %v want %v", got, id)
	}
}

func TestParseCitizenChannel_rejectsUnknownPrefix(t *testing.T) {
	_, err := ParseCitizenChannel("other:channel")
	if err == nil {
		t.Fatal("expected error")
	}
}
