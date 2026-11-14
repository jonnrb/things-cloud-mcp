package main

import (
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
	"github.com/google/uuid"
)

func TestGenerateUUIDIsCanonical(t *testing.T) {
	for i := 0; i < 5000; i++ {
		id := generateUUID()
		if err := thingscloud.ValidateUUID(id); err != nil {
			t.Fatalf("generateUUID() = %q is not a valid Things ID: %v", id, err)
		}
	}
}

func TestEncodeUUIDKeepsLeadingZeroByte(t *testing.T) {
	var u uuid.UUID
	u[0] = 0
	for i := 1; i < len(u); i++ {
		u[i] = 0xff
	}
	got := thingscloud.EncodeUUID(u)
	if got[0] != '1' {
		t.Fatalf("EncodeUUID with leading zero byte = %q, want leading '1'", got)
	}
}
