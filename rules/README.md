# Rule packs

YAML rule packs for source, library, and configuration detection live here.
Schema and invariants are documented in [DESIGN.md](../DESIGN.md) §5.

## libraries/

`catalog.yaml` lists known cryptographic dependencies for the `deps` detector.
A match is **medium** confidence at most — presence in a lockfile is not proof of use.
