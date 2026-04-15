####
##  Dependency versions
####

# renovate: datasource=github-releases depName=golangci/golangci-lint versioning=semver
GOLANGCI_LINT_VERSION := 2.11.4

# renovate: datasource=go depName=github.com/goph/licensei versioning=semver
LICENSEI_VERSION = 0.9.0

BIN := ${PWD}/bin

export PATH := ${BIN}:${PATH}

GOVERSION := $(shell go env GOVERSION)

LICENSEI := ${BIN}/licensei

GOLANGCI_LINT := ${BIN}/golangci-lint
LINTER_FLAGS := --timeout 10m

## =============
## ==  Rules  ==
## =============

.PHONY: check
check: license-check lint test

.PHONY: license-check
license-check: ${LICENSEI} .licensei.cache ## Run license check
	${LICENSEI} check
	${LICENSEI} header

.PHONY: license-cache
license-cache: ${LICENSEI} ## Generate license cache
	${LICENSEI} cache

.PHONY: lint
lint: ${GOLANGCI_LINT} ## Run linter
	${GOLANGCI_LINT} run ${LINTER_FLAGS}

.PHONY: lint-fix
lint-fix: ${GOLANGCI_LINT} ## Run linter
	${GOLANGCI_LINT} run --fix

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: tidy
tidy: ## Tidy Go modules
	find . -iname "go.mod" -not -path "./.devcontainer/*" | xargs -L1 sh -c 'cd $$(dirname $$0); go mod tidy'

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

## =========================
## ==  Tool dependencies  ==
## =========================

${GOLANGCI_LINT}: ${GOLANGCI_LINT}_${GOLANGCI_LINT_VERSION}_${GOVERSION} | ${BIN}
	ln -sf $(notdir $<) $@

${GOLANGCI_LINT}_${GOLANGCI_LINT_VERSION}_${GOVERSION}: IMPORT_PATH := github.com/golangci/golangci-lint/v2/cmd/golangci-lint
${GOLANGCI_LINT}_${GOLANGCI_LINT_VERSION}_${GOVERSION}: VERSION := v${GOLANGCI_LINT_VERSION}
${GOLANGCI_LINT}_${GOLANGCI_LINT_VERSION}_${GOVERSION}: | ${BIN}
	${go_install_binary}

${LICENSEI}: ${LICENSEI}_${LICENSEI_VERSION}_${GOVERSION} | ${BIN}
	ln -sf $(notdir $<) $@

${LICENSEI}_${LICENSEI_VERSION}_${GOVERSION}: IMPORT_PATH := github.com/goph/licensei/cmd/licensei
${LICENSEI}_${LICENSEI_VERSION}_${GOVERSION}: VERSION := v${LICENSEI_VERSION}
${LICENSEI}_${LICENSEI_VERSION}_${GOVERSION}: | ${BIN}
	${go_install_binary}

.licensei.cache: ${LICENSEI}
ifndef GITHUB_TOKEN
	@>&2 echo "WARNING: building licensei cache without Github token, rate limiting might occur."
	@>&2 echo "(Hint: If too many licenses are missing, try specifying a Github token via the environment variable GITHUB_TOKEN.)"
endif
	${LICENSEI} cache

${BIN}:
	mkdir -p ${BIN}

define go_install_binary
find ${BIN} -name '$(notdir ${IMPORT_PATH})_*' -exec rm {} +
GOBIN=${BIN} go install ${IMPORT_PATH}@${VERSION}
mv ${BIN}/$(notdir ${IMPORT_PATH}) $@
endef

# Self-documenting Makefile
.DEFAULT_GOAL = help
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
