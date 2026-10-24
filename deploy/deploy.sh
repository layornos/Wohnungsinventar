#!/usr/bin/env bash
# Builds the Pi binary, uploads it and restarts the service (see README, "Betrieb auf dem Pi").
# The systemd units and inventar.env are installed once by hand; this script only updates the binary.
#
#   deploy/deploy.sh
#   INVENTAR_DEPLOY_HOST=layornos@192.168.178.87 deploy/deploy.sh
set -euo pipefail

host="${INVENTAR_DEPLOY_HOST:-layornos@kodi}"
remote_dir="${INVENTAR_DEPLOY_DIR:-/home/layornos/inventar}"
service="inventar.service"

cd "$(dirname "$0")/.."

make build-pi

# Upload under a temporary name and move it into place, so the running binary is never half written.
ssh "$host" "mkdir -p '$remote_dir/bin' '$remote_dir/data'"
scp bin/inventar-linux-arm64 "$host:$remote_dir/bin/.inventar.new"
ssh "$host" "chmod 755 '$remote_dir/bin/.inventar.new' && mv -f '$remote_dir/bin/.inventar.new' '$remote_dir/bin/inventar'"

# -t so sudo can ask for the password.
ssh -t "$host" "sudo systemctl restart $service && systemctl is-active $service"
echo "Deployed to $host:$remote_dir/bin/inventar"
