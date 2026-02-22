.PHONY: help build-local test prod deploy ping-test ping-prod logs-test logs-prod logs

PLAYBOOK ?= deploy_nexo_im.yml
INVENTORY ?=
ANSIBLE_PLAYBOOK ?= ansible-playbook
ANSIBLE ?= ansible
LIMIT ?= nexo
EXTRA ?=
LOG_ROOT ?= /opt/nexo_im/shared/logs
LOG_DEST ?= ./logs
GO ?= go
TARGET_GOOS ?= linux
TARGET_GOARCH ?= amd64
TARGET_CGO_ENABLED ?= 0
PROJECT_ROOT ?= $(abspath $(CURDIR)/..)
LOCAL_BUILD_DIR ?= $(CURDIR)/.build
LOCAL_BUILD_BIN ?= $(LOCAL_BUILD_DIR)/nexo_im-$(TARGET_GOOS)-$(TARGET_GOARCH)

help:
	@echo "Usage:"
	@echo "  make build-local       # build linux/amd64 binary on local machine"
	@echo "  make test              # deploy with inventory/test.ini"
	@echo "  make prod              # deploy with inventory/prod.ini"
	@echo "  make deploy INVENTORY=inventory/test.ini"
	@echo "  make ping-test         # connectivity check for test hosts"
	@echo "  make ping-prod         # connectivity check for prod hosts"
	@echo "  make logs-test         # fetch app logs with inventory/test.ini"
	@echo "  make logs-prod         # fetch app logs with inventory/prod.ini"
	@echo "  make logs INVENTORY=inventory/test.ini LOG_DEST=./logs/test"
	@echo "  make deploy INVENTORY=... EXTRA='--private-key ~/.ssh/xxx.pem'"

build-local:
	@mkdir -p "$(LOCAL_BUILD_DIR)"
	cd "$(PROJECT_ROOT)" && GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) CGO_ENABLED=$(TARGET_CGO_ENABLED) $(GO) build -o "$(LOCAL_BUILD_BIN)" ./cmd/server

test: build-local
	$(ANSIBLE_PLAYBOOK) -i inventory/test.ini $(PLAYBOOK) -e "local_build_bin=$(LOCAL_BUILD_BIN)" -e "infra_env=TEST" $(EXTRA)

prod: build-local
	$(ANSIBLE_PLAYBOOK) -i inventory/prod.ini $(PLAYBOOK) -e "local_build_bin=$(LOCAL_BUILD_BIN)" -e "infra_env=PROD" $(EXTRA)

deploy: build-local
	@test -n "$(INVENTORY)" || (echo "INVENTORY is required, e.g. make deploy INVENTORY=inventory/test.ini"; exit 1)
	$(ANSIBLE_PLAYBOOK) -i $(INVENTORY) $(PLAYBOOK) -e "local_build_bin=$(LOCAL_BUILD_BIN)" $(EXTRA)

ping-test:
	$(ANSIBLE) -i inventory/test.ini $(LIMIT) -m ping $(EXTRA)

ping-prod:
	$(ANSIBLE) -i inventory/prod.ini $(LIMIT) -m ping $(EXTRA)

logs-test:
	$(MAKE) logs INVENTORY=inventory/test.ini LOG_DEST=./logs/test EXTRA="$(EXTRA)"

logs-prod:
	$(MAKE) logs INVENTORY=inventory/prod.ini LOG_DEST=./logs/prod EXTRA="$(EXTRA)"

logs:
	@test -n "$(INVENTORY)" || (echo "INVENTORY is required, e.g. make logs INVENTORY=inventory/test.ini"; exit 1)
	@mkdir -p "$(LOG_DEST)"
	$(ANSIBLE) -i $(INVENTORY) $(LIMIT) -b -m ansible.builtin.fetch -a "src=$(LOG_ROOT)/nexo_im.out.log dest=$(LOG_DEST)/ flat=no" $(EXTRA)
	$(ANSIBLE) -i $(INVENTORY) $(LIMIT) -b -m ansible.builtin.fetch -a "src=$(LOG_ROOT)/nexo_im.err.log dest=$(LOG_DEST)/ flat=no" $(EXTRA)
