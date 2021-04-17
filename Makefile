dockerize := ./scripts/dockerize.sh
scripts := $(wildcard scripts/*.sh)

all:
	@echo "Use 'make \$$target' to build a container for the desired target."
	@echo "Run the container directly, e.g. 'docker run --rm msgbounds:\$$target -h'."

check-scripts:
	shellcheck -x -P scripts $(scripts)

check-tester:
	golangci-lint run

check: check-scripts check-tester

lightftp: ; $(dockerize) $@
pure-ftpd: ; $(dockerize) $@
