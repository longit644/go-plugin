package shared

import "crypto/rand"

func RandBytes(size int) []byte {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)

	return buf
}
