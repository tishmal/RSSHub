package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type UUID [16]byte

func NewUUID() (UUID, error) {
	var uuid UUID
	_, err := rand.Read(uuid[:])
	if err != nil {
		return UUID{}, fmt.Errorf("failed to generate UUID: %w", err)
	}
	uuid[6] &= 0x0F
	uuid[6] |= 0x40
	uuid[8] &= 0x3F
	uuid[8] |= 0x80
	return uuid, nil
}

func (u UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func (u UUID) IsZero() bool {
	for _, b := range u[:] {
		if b != 0 {
			return false
		}
	}
	return true
}

func ParseUUID(s string) (UUID, error) {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return UUID{}, fmt.Errorf("invalid UUID length: got %d, expected 32", len(s))
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return UUID{}, fmt.Errorf("failed to decode UUID: %w", err)
	}

	var uuid UUID
	copy(uuid[:], bytes)
	return uuid, nil
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseUUID(s)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}
