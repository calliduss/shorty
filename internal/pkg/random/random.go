package random

import (
	"math/rand"
	"time"
)

func GenerateRandomString(length int) string {
	charset := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	seed := rand.NewSource(time.Now().UTC().UnixNano())
	random := rand.New(seed)

	result := make([]rune, length)
	for i := range result {
		result[i] = charset[random.Intn(len(charset))]
	}

	return string(result)
}
