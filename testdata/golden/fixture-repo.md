# cryptarium report

Tool: cryptarium 0.0.0-dev

Findings: **4** · 2 critical · 2 high · 0 medium · 0 low

| Priority | Class | Location | Primitive | Recommendation |
|---|---|---|---|---|
| critical (83) | broken | `certs/ecdsa-p256.pem` | ECDSA | ML-DSA-65 (FIPS 204) |
| critical (83) | broken | `certs/rsa2048.pem` | RSA | ML-KEM-768 (key establishment) / ML-DSA-65 (signatures) (FIPS 203 / FIPS 204) |
| high (70) | broken | `go.mod:6` | RSA | ML-KEM-768 (key establishment) / ML-DSA-65 (signatures) (FIPS 203 / FIPS 204) |
| high (78) | broken | `token.go:9` | RSA | ML-KEM-768 (key establishment) / ML-DSA-65 (signatures) (FIPS 203 / FIPS 204) |

Scores use vulnerability, longevity, exposure, and agility heuristics (DESIGN.md §7).
