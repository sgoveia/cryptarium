# Rule packs

YAML rule packs for source, library, and configuration detection live here.
Schema and invariants are documented in [DESIGN.md](../DESIGN.md) §5.

Release binaries and `go install` builds embed these packs (`embed.go`) so scans work without a source checkout. When an on-disk `rules/` tree is found by walking up from the working directory, it takes precedence over the embed. Edit here, then rebuild only when you want the binary to pick up the change permanently.

## Source packs

| Directory | Packs |
|---|---|
| `go/` | Go stdlib crypto |
| `python/` | `cryptography`, hashlib |
| `javascript/` | Web Crypto / Node crypto (also applies to TypeScript) |
| `java/` | `java.security` |
| `c/` | OpenSSL, libsodium, mbedTLS (also applied to C++ files) |
| `cpp/` | Crypto++, Botan (C++-only identifiers) |

## libraries/

`catalog.yaml` lists known cryptographic dependencies for the `deps` detector.
A match is **medium** confidence at most (presence in a lockfile is not proof of use).
