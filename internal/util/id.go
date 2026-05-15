package util

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

const roomCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const roomCodeLen = 6

func GenerateRoomCode() string {
	b := make([]byte, roomCodeLen)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(roomCodeChars))))
		b[i] = roomCodeChars[n.Int64()]
	}
	return string(b)
}

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func NormalizeRoomCode(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "O", "0")
	s = strings.ReplaceAll(s, "I", "1")
	s = strings.ReplaceAll(s, "L", "1")
	return s
}
