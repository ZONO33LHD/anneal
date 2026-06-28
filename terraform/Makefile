GIT_PREFIX := $(shell git rev-parse --show-prefix)

terraform-init: ## Use this like `make terraform-init PREFIX_DIR=$(git rev-parse --show-prefix)`
ifndef PREFIX_DIR
	@echo "PREFIX_DIR is required"
	exit 1
endif
	$(eval TFSTATE_PREFIX := $(shell echo $(PREFIX_DIR) | sed -e 's/\/$$//' | sed -e 's/\//./g'))
	$(eval TERRAFORM_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST)))))
	cp $(TERRAFORM_DIR)/init.tf $(PWD)
	terraform init -reconfigure -backend-config "prefix=$(TFSTATE_PREFIX)"
.PHONY: terraform-init
