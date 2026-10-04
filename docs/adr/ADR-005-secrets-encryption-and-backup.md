# ADR-005: AES-256-GCM Secrets Encryption and Idempotent Backup/Restore

## Status
Accepted

## Date
2026-10-04

## Context
Self-hosted smart home hubs store sensitive information such as integration tokens, Wi-Fi credentials, and MQTT authentication details. 
Dumping raw SQLite databases to disk or archiving files in unencrypted formats exposes private credentials to anyone with physical access to backup files or shared network drives.
Conversely, relying on external secret management vaults (e.g. HashiCorp Vault, AWS KMS) violates the core requirement of zero external cloud dependencies and low RAM consumption on single-board computers.

## Decision
We implement a zero-dependency, local cryptographic secret management and backup engine:
1. **Symmetric Encryption**: Secrets and backup archives are encrypted using authenticated **AES-256-GCM**.
2. **Key Derivation**: Passphrases generate cryptographic 256-bit keys via **PBKDF2-HMAC-SHA256** using 100,000 iterations and cryptographically secure random 16-byte salts.
3. **Backup Format**:
   - Plain backups produce standard `.tar.gz` archives containing `domus.db`, `domus.toml`, and a signed `backup_manifest.json`.
   - Encrypted backups produce `.tar.gz.enc` where the entire archive is encrypted via AES-256-GCM with salt and nonce prepended.
4. **Idempotent Restore & Security Hardening**:
   - The CLI (`domus-cli restore`) validates tar archive headers against path-traversal attacks (e.g., zip-slip) and ensures write targets stay strictly within configured destination directories.
   - Restores are atomic and safe across restarts.

## Consequences
### Positive
- Military-grade confidentiality and integrity verification without external services.
- Protection against tampering via GCM authentication tags.
- Simple, standardized administration via `domus-cli backup` and `domus-cli restore`.

### Negative
- If a user loses their encryption passphrase, encrypted backups cannot be recovered.
