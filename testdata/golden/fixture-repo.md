# cryptarium report

Tool: cryptarium 0.0.0-dev

Findings: **3**

| Priority signal | Location | Primitive | Evidence |
|---|---|---|---|
| high · certificate | `certs/ecdsa-p256.pem` | ECDSA | x509 certificate CN=cryptarium-fixture-p256; ECDSA P-256 |
| high · certificate | `certs/rsa2048.pem` | RSA | x509 certificate CN=cryptarium-fixture-rsa2048; RSA-2048 |
| medium · dependency | `go.mod:6` | RSA | require golang.org/x/crypto v0.31.0 |

Confidence reflects detection strength only. Classification and risk scoring land in later phases.
