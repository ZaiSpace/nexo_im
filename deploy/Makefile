.PHONY: help test prod deploy ping-test ping-prod logs-test logs-prod logs

PLAYBOOK ?= deploy_nexo_im.yml
INVENTORY ?=
ANSIBLE_PLAYBOOK ?= ansible-playbook
ANSIBLE ?= ansible
LIMIT ?= nexo
EXTRA ?=
LOG_ROOT ?= /opt/nexo_im/shared/logs
LOG_DEST ?= ./logs

help:
	@echo "Usage:"
	@echo "  make test              # deploy with inventory/test.ini"
	@echo "  make prod              # deploy with inventory/prod.ini"
	@echo "  make deploy INVENTORY=inventory/test.ini"
	@echo "  make ping-test         # connectivity check for test hosts"
	@echo "  make ping-prod         # connectivity check for prod hosts"
	@echo "  make logs-test         # fetch app logs with inventory/test.ini"
	@echo "  make logs-prod         # fetch app logs with inventory/prod.ini"
	@echo "  make logs INVENTORY=inventory/test.ini LOG_DEST=./logs/test"
	@echo "  make deploy INVENTORY=... EXTRA='--private-key ~/.ssh/xxx.pem'"

test:
	$(ANSIBLE_PLAYBOOK) -i inventory/test.ini $(PLAYBOOK) $(EXTRA)

prod:
	$(ANSIBLE_PLAYBOOK) -i inventory/prod.ini $(PLAYBOOK) $(EXTRA)

deploy:
	@test -n "$(INVENTORY)" || (echo "INVENTORY is required, e.g. make deploy INVENTORY=inventory/test.ini"; exit 1)
	$(ANSIBLE_PLAYBOOK) -i $(INVENTORY) $(PLAYBOOK) $(EXTRA)

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
