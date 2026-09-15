package positive

import (
	"crypto/rand"
	"crypto/rsa"
)

func F() { rsa.GenerateKey(rand.Reader, 2048) }
