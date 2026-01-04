package common

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt returns a random integer between min and max (inclusive).
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min+1) + min
}
