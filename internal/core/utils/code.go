package utils

import (
	"crypto/rand"
)

// RandomInviteCode returns a length-n uppercase alphanumeric code without
// visually-confusing characters (no 0/O/1/I).
func RandomInviteCode(n int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range buf {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

// Itoa is a tiny positive-int formatter used by dynamic SQL builders when
// they need "$1", "$2", … placeholders — avoids pulling in strconv.
func Itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
