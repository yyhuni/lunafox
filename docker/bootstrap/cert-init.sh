#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${PUBLIC_HOST:?PUBLIC_HOST is required}"
[[ "$PUBLIC_HOST" =~ ^[a-zA-Z0-9.:-]+$ ]] || {
	echo 'PUBLIC_HOST must be a hostname or IP address' >&2
	exit 1
}
certificate_directory="${LUNAFOX_CERTIFICATE_DIRECTORY:-/etc/nginx/ssl}"
[[ "$certificate_directory" = /* ]] || {
	echo 'LUNAFOX_CERTIFICATE_DIRECTORY must be absolute' >&2
	exit 1
}
mkdir -p "$certificate_directory"
cd "$certificate_directory"
san="DNS:$PUBLIC_HOST"
if [[ "$PUBLIC_HOST" == *:* || "$PUBLIC_HOST" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	san="IP:$PUBLIC_HOST"
fi
if [[ -e fullchain.pem || -e privkey.pem || -L fullchain.pem || -L privkey.pem ]]; then
	[[ -f fullchain.pem && ! -L fullchain.pem && -f privkey.pem && ! -L privkey.pem ]] || {
		echo 'Incomplete certificate state; explicit repair required' >&2
		exit 1
	}
	file_mode() {
		local mode
		mode="$(stat -c '%a' "$1" 2>/dev/null || true)"
		if [[ -z "$mode" ]]; then
			mode="$(stat -f '%Lp' "$1")"
		fi
		printf '%s' "$mode"
	}
	[[ "$(file_mode fullchain.pem)" = 600 && "$(file_mode privkey.pem)" = 600 ]] || {
		echo 'Certificate files must have mode 0600' >&2
		exit 1
	}
	openssl x509 -in fullchain.pem -noout -checkend 0
	openssl x509 -in fullchain.pem -noout -text | awk -v expected="$san" '
		/X509v3 Subject Alternative Name:/ {
			getline
			n = split($0, entries, ",")
			for (i = 1; i <= n; i++) {
				gsub(/^[[:space:]]+|[[:space:]]+$/, "", entries[i])
				if (entries[i] == expected) found = 1
			}
		}
		END { exit !found }
	' || {
		echo 'Certificate SAN does not match PUBLIC_HOST' >&2
		exit 1
	}
	cert_key=$(openssl x509 -in fullchain.pem -pubkey -noout)
	private_key=$(openssl pkey -in privkey.pem -pubout)
	[[ "$cert_key" == "$private_key" ]] || {
		echo 'Certificate and private key do not match' >&2
		exit 1
	}
	exit 0
fi
# Keep partial generation separate. A partial published pair is rejected on retry.
work=$(mktemp -d .cert-init.XXXXXX)
trap 'rm -rf -- "$work"' EXIT
openssl req -x509 -newkey rsa:2048 -nodes -days 365 -subj "/CN=$PUBLIC_HOST" -addext "subjectAltName=$san" -keyout "$work/privkey.pem" -out "$work/fullchain.pem"
ln "$work/privkey.pem" privkey.pem
ln "$work/fullchain.pem" fullchain.pem
