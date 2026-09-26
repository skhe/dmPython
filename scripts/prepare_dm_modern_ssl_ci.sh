#!/usr/bin/env bash
set -euo pipefail

root="${DM_SSL_MODERN_ROOT:-$PWD/ssl-modern}"
container="${DM_SSL_TEST_CONTAINER:-dmpython-ci-dm8-ssl}"
user="${DM_SSL_TEST_USER:-SYSDBA}"
umask 077
mkdir -p "$root/server" "$root/client"

openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 2 \
  -keyout "$root/ca-key.pem" -out "$root/ca-cert.pem" \
  -subj '/CN=dmPython CI Test CA' \
  -addext 'basicConstraints=critical,CA:TRUE' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign' >/dev/null 2>&1
openssl req -newkey rsa:2048 -nodes -sha256 \
  -keyout "$root/server/server-key.pem" -out "$root/server.csr" \
  -subj '/CN=server' >/dev/null 2>&1
cat > "$root/server.ext" <<'EOF'
subjectAltName=IP:127.0.0.1
extendedKeyUsage=serverAuth
keyUsage=digitalSignature,keyEncipherment
EOF
openssl x509 -req -in "$root/server.csr" \
  -CA "$root/ca-cert.pem" -CAkey "$root/ca-key.pem" -CAcreateserial \
  -out "$root/server/server-cert.pem" -days 2 -sha256 \
  -extfile "$root/server.ext" >/dev/null 2>&1

openssl req -newkey rsa:2048 -nodes -sha256 \
  -keyout "$root/client/client-key.pem" -out "$root/client.csr" \
  -subj "/CN=$user" >/dev/null 2>&1
cat > "$root/client.ext" <<'EOF'
extendedKeyUsage=clientAuth
keyUsage=digitalSignature,keyEncipherment
EOF
openssl x509 -req -in "$root/client.csr" \
  -CA "$root/ca-cert.pem" -CAkey "$root/ca-key.pem" -CAcreateserial \
  -out "$root/client/client-cert.pem" -days 2 -sha256 \
  -extfile "$root/client.ext" >/dev/null 2>&1

cp "$root/ca-cert.pem" "$root/server/ca-cert.pem"
cp "$root/ca-cert.pem" "$root/client/ca-cert.pem"
docker cp "$root/server/." "$container:/opt/dmdbms/bin/server_ssl/"
docker cp "$root/client/." "$container:/opt/dmdbms/bin/client_ssl/$user/"
docker exec "$container" chown -R dmdba:dinstall \
  /opt/dmdbms/bin/server_ssl "/opt/dmdbms/bin/client_ssl/$user"
docker restart "$container" >/dev/null
echo "DM8 restarted with short-lived SAN certificate"
