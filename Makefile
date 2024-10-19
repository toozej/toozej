# Set sane defaults for Make
SHELL = bash
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# Set default goal such that `make` runs `make help`
.DEFAULT_GOAL := help

OS = $(shell uname -s)
ifeq ($(OS), Linux)
	OPENER=xdg-open
else
	OPENER=open
endif

.PHONY: all run serve kill clean help

all: clean run kill serve ## Run default workflow via Docker

run: ## Run built Docker image
	docker run --rm --name github-profile-generator --env-file $(CURDIR)/.env -v $(CURDIR)/:/tmp/gpg/ ghcr.io/charmbracelet/markscribe:latest -write /tmp/gpg/README.md  /tmp/gpg/templates/README.md.tpl

serve: run kill ## Serve the built README.md in a markdown web-server
	docker run -d --rm --name github-profile-server -p 9000:3080 -v $(CURDIR)/README.md:/data/README.md thomsch98/markserv
	$(OPENER) http://localhost:9000/README.md

kill: ## Kill any already-running github-profile-server container
	docker kill github-profile-server

clean: ## Remove any locally built README
	rm -f $(CURDIR)/README.md

help: ## Display help text
	@grep -E '^[a-zA-Z_-]+ ?:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
