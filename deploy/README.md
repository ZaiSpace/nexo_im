# Deploy with Ansible + Supervisor

This directory contains a deployment setup for `nexo_im` on Debian servers.

It is designed to run on the same EC2 instance as `island` without conflict:

- independent deploy root: `/opt/nexo_im`
- independent supervisor program: `nexo_im`
- independent supervisor conf: `/etc/supervisor/conf.d/nexo_im.conf`

## 1) Initialize a new Debian EC2 host

SSH to target host and run:

```bash
sudo bash deploy/scripts/install_debian.sh
```

Optional Go version:

```bash
sudo bash deploy/scripts/install_debian.sh 1.25.6
```

## 2) Prepare inventory (TEST / PROD)

From `deploy/`:

```bash
cp inventory/test.ini.example inventory/test.ini
cp inventory/prod.ini.example inventory/prod.ini
```

Edit host/user in each file.

Environment is set by inventory vars:

- `inventory/test.ini`: `infra_env=TEST`
- `inventory/prod.ini`: `infra_env=PROD`

## 3) Run deployment

From `deploy/`:

```bash
# Test environment
make test

# Production environment
make prod
```

With SSH key:

```bash
make test EXTRA='--private-key ~/.ssh/your-key.pem'
make prod EXTRA='--private-key ~/.ssh/your-key.pem'
```

Custom inventory:

```bash
make deploy INVENTORY=inventory/test.ini
```

## Optional overrides

Common defaults live in `group_vars/all.yml`:

- `deploy_root` (default `/opt/nexo_im`)
- `run_user` (default `www-data`)
- `supervisor_conf_path` (default `/etc/supervisor/conf.d/nexo_im.conf`)

## Notes

- Build runs on the target host: `go build -o bin/nexo_im ./cmd/server`.
- Runtime environment is selected by `INFRA_ENV` (LOCAL/TEST/PROD).
- Logs are written to `{{ shared_root }}/logs` on the target host.
