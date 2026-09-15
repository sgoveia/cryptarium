package fixture

import (
	"crypto/rand"
	"crypto/rsa"
)

func makeTokenKey() {
	rsa.GenerateKey(rand.Reader, 2048)
}
