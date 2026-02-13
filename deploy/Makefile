.PHONY: help test prod deploy ping-test ping-prod

PLAYBOOK ?= deploy_nexo_im.yml
INVENTORY ?=
ANSIBLE_PLAYBOOK ?= ansible-playbook
ANSIBLE ?= ansible
LIMIT ?= nexo
EXTRA ?=

help:
	@echo "Usage:"
	@echo "  make test              # deploy with inventory/test.ini"
	@echo "  make prod              # deploy with inventory/prod.ini"
	@echo "  make deploy INVENTORY=inventory/test.ini"
	@echo "  make ping-test         # connectivity check for test hosts"
	@echo "  make ping-prod         # connectivity check for prod hosts"
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
