# cryptarium report

Tool: cryptarium 0.0.0-dev

Findings: **4**

| Class | Location | Primitive | Evidence |
|---|---|---|---|
| broken | `certs/ecdsa-p256.pem` | ECDSA | x509 certificate CN=cryptarium-fixture-p256; ECDSA P-256 |
| broken | `certs/rsa2048.pem` | RSA | x509 certificate CN=cryptarium-fixture-rsa2048; RSA-2048 |
| broken | `go.mod:6` | RSA | require golang.org/x/crypto v0.31.0 |
| broken | `token.go:9` | RSA | rsa.GenerateKey(rand.Reader, 2048) |

Confidence reflects detection strength only. Classification and risk scoring land in later phases.
