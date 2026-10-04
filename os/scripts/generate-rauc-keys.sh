#!/usr/bin/env bash
# ─── Domus OS - Generate RAUC x.509 PKI Signing Keys ──────────────────────────
# Generates a Certificate Authority (CA) and a release signing certificate/key
# for signing and verifying atomic OTA / offline update bundles (.raucb).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="${SCRIPT_DIR}/../rauc/certs"
mkdir -p "${CERTS_DIR}"

DAYS=3650
KEY_SIZE=4096

echo "=== Generating Domus OS RAUC x.509 PKI in ${CERTS_DIR} ==="

# 1. Generate Master Certificate Authority (CA) Private Key
openssl genrsa -out "${CERTS_DIR}/ca.key.pem" ${KEY_SIZE}

# 2. Generate Master CA Certificate
openssl req -x509 -new -nodes \
    -key "${CERTS_DIR}/ca.key.pem" \
    -sha256 -days ${DAYS} \
    -out "${CERTS_DIR}/ca.cert.pem" \
    -subj "/O=Domus Hub/CN=Domus OS Release CA"

# 3. Generate Development Release Signing Key
openssl genrsa -out "${CERTS_DIR}/development.key.pem" ${KEY_SIZE}

# 4. Generate Certificate Signing Request (CSR)
openssl req -new \
    -key "${CERTS_DIR}/development.key.pem" \
    -out "${CERTS_DIR}/development.csr.pem" \
    -subj "/O=Domus Hub/CN=Domus OS Development Signer"

# 5. Sign Development Certificate using the Master CA
cat << 'EOF' > "${CERTS_DIR}/rauc-ext.cnf"
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature
extendedKeyUsage = critical, 1.3.6.1.4.1.53127.1.1
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
EOF

openssl x509 -req \
    -in "${CERTS_DIR}/development.csr.pem" \
    -CA "${CERTS_DIR}/ca.cert.pem" \
    -CAkey "${CERTS_DIR}/ca.key.pem" \
    -CAcreateserial \
    -out "${CERTS_DIR}/development.cert.pem" \
    -days ${DAYS} \
    -sha256 \
    -extfile "${CERTS_DIR}/rauc-ext.cnf"

# Cleanup temporary CSR and config
rm -f "${CERTS_DIR}/development.csr.pem" "${CERTS_DIR}/rauc-ext.cnf" "${CERTS_DIR}/ca.cert.srl"

echo "=== RAUC Keys Generated Successfully ==="
echo "  CA Certificate:   ${CERTS_DIR}/ca.cert.pem"
echo "  Signing Cert:     ${CERTS_DIR}/development.cert.pem"
echo "  Signing Key:      ${CERTS_DIR}/development.key.pem"
