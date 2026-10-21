#!/bin/sh
# Local testing only. Does not issue a publicly trusted certificate.
set -eu
cd "$(dirname "$0")/.."
certificate_dir=local-certs/live/localhost
if [ -e "$certificate_dir/fullchain.pem" ] || [ -e "$certificate_dir/privkey.pem" ]; then
    if [ ! -f "$certificate_dir/fullchain.pem" ] || [ ! -f "$certificate_dir/privkey.pem" ]; then
        echo "Incomplete certificate in $certificate_dir; leaving existing files unchanged." >&2
        exit 1
    fi
    if ! openssl x509 -in "$certificate_dir/fullchain.pem" -noout -checkend 0; then
        echo "Local certificate is invalid or expired; remove the local certificate files before regenerating." >&2
        exit 1
    fi
    echo "Local certificate already exists in $certificate_dir; leaving it unchanged."
    exit 0
fi
umask 077
mkdir -p "$certificate_dir"
openssl req -x509 -newkey rsa:2048 -sha256 -nodes -days 30 \
    -subj '/CN=localhost' \
    -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1' \
    -keyout "$certificate_dir/privkey.pem" \
    -out "$certificate_dir/fullchain.pem"
chmod 644 "$certificate_dir/fullchain.pem"
echo "Created local certificate. Set DOMAIN=localhost and TLS_DIRECTORY=./local-certs in .env."
