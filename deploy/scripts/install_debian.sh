#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root: sudo bash $0 [go_version]"
  exit 1
fi

GO_VERSION="${1:-1.25.6}"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  git \
  rsync \
  supervisor \
  tar

curl -fsSL "${GO_URL}" -o "/tmp/${GO_TARBALL}"
rm -rf /usr/local/go
tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"

cat >/etc/profile.d/go.sh <<'EOS'
export PATH=$PATH:/usr/local/go/bin
EOS
chmod 0644 /etc/profile.d/go.sh

if ! grep -q '/usr/local/go/bin' /etc/environment; then
  echo 'PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin"' >/etc/environment
fi

systemctl enable supervisor
systemctl restart supervisor

echo "Installed successfully:"
/usr/local/go/bin/go version
supervisorctl version || true
echo "Re-login or run: source /etc/profile.d/go.sh"
