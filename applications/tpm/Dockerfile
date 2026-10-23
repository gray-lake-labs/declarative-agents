# syntax=docker/dockerfile:1
#
# The image each scrum-master release publishes (eng03): tpm at the tag, the
# declarative-agents runtime at the release tpm pins, and the spec tool built
# at that release, so a CI job runs tpm and a profile with no install step
# and no Go toolchain. The build context is the directory mage publish
# builds the release binaries into; the image copies the binary for its
# architecture, so its tpm is the one on the GitHub release.

ARG GO_IMAGE=golang:1.26-bookworm
ARG BASE_IMAGE=debian:bookworm-slim

# The runtime, fetched and verified by tpm install --runtime, and the spec
# tool, which it builds with Go here so the final image needs none.
FROM ${GO_IMAGE} AS runtime
ARG TARGETARCH
ENV XDG_CACHE_HOME=/opt/tpm/cache
COPY tpm_linux_${TARGETARCH} /usr/local/bin/tpm
RUN tpm install --runtime \
 && ln -s "$(tpm install --runtime --json | sed -n 's/^ *"agent": "\(.*\)",$/\1/p')" /opt/tpm/agent

FROM ${BASE_IMAGE}
ARG TARGETARCH
ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.title="tpm" \
      org.opencontainers.image.description="tpm and the declarative-agents runtime it pins" \
      org.opencontainers.image.source="https://github.com/gray-lake-labs/scrum-master" \
      org.opencontainers.image.vendor="Gray Lake Labs" \
      org.opencontainers.image.licenses="MIT-0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates git git-crypt \
 && rm -rf /var/lib/apt/lists/* \
 && git config --system safe.directory '*' \
 && useradd --uid 1000 --create-home --shell /bin/sh tpm \
 && install -d -o tpm -g tpm /work
COPY --from=runtime --chown=tpm:tpm /opt/tpm /opt/tpm
COPY tpm_linux_${TARGETARCH} /usr/local/bin/tpm
RUN ln -s "$(readlink /opt/tpm/agent)" /usr/local/bin/agent
ENV XDG_CACHE_HOME=/opt/tpm/cache
USER tpm
WORKDIR /work
ENTRYPOINT []
CMD ["tpm", "help"]
