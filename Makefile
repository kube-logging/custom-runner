
BIN := ${PWD}/bin
export PATH := ${BIN}:${PATH}

# renovate: datasource=go depName=github.com/goph/licensei versioning=semver
LICENSEI_VERSION = 0.9.0

LICENSEI := ${BIN}/licensei

${BIN}:
	mkdir -p ${BIN}

.PHONY: license-check
license-check: ${LICENSEI} ## Run license check
	${LICENSEI} check
	${LICENSEI} header

.PHONY: license-cache
license-cache: ${LICENSEI} ## Generate license cache
	${LICENSEI} cache

.PHONY: check
check: license-cache license-check

${LICENSEI}: ${LICENSEI}_v${LICENSEI_VERSION} | ${BIN}
	ln -sf $(notdir $<) $@

${LICENSEI}_v${LICENSEI_VERSION}: IMPORT_PATH := github.com/goph/licensei/cmd/licensei
${LICENSEI}_v${LICENSEI_VERSION}: VERSION := v${LICENSEI_VERSION}
${LICENSEI}_v${LICENSEI_VERSION}: | ${BIN}
	${go_install_binary}

define go_install_binary
find ${BIN} -name '$(notdir ${IMPORT_PATH})_*' -exec rm {} +
GOBIN=${BIN} go install ${IMPORT_PATH}@${VERSION}
mv ${BIN}/$(notdir ${IMPORT_PATH}) ${BIN}/$(notdir ${IMPORT_PATH})_${VERSION}
endef
