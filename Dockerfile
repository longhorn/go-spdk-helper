# syntax=docker/dockerfile:1.22.0
FROM registry.suse.com/bci/bci-base:16.0 AS base

ARG TARGETARCH
ARG http_proxy
ARG https_proxy
ARG SRC_BRANCH=master
ARG SRC_TAG
ARG CACHEBUST

ENV GOLANG_VERSION=1.26.1
ENV GOLANGCI_LINT_VERSION=v2.11.4

ENV ARCH=${TARGETARCH}
ENV GOFLAGS=-mod=vendor

RUN for i in {1..10}; do \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/system:/snappy/openSUSE_Factory/system:snappy.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/network:/utilities/16.0/network:utilities.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/devel:libraries:c_c++/16.0/devel:libraries:c_c++.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/devel:/languages:/python:/Factory/16.0/devel:languages:python:Factory.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/devel:languages:python:backports/16.0/devel:languages:python:backports.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/CrossToolchain:avr/16.0/CrossToolchain:avr.repo && \
        zypper -n addrepo --refresh https://download.opensuse.org/repositories/devel:tools:compiler/16.0/devel:tools:compiler.repo && \
        zypper --gpg-auto-import-keys ref && break || sleep 1; \
    done

RUN zypper -n install cmake gcc gcc13 unzip tar xsltproc docbook-xsl-stylesheets python3 python3-pip fuse3-devel \
              e2fsprogs xfsprogs util-linux-systemd libcmocka-devel device-mapper procps jq git

# Install Go
ENV GOPATH=/go PATH=/go/bin:/usr/local/go/bin:${PATH} SHELL=/bin/bash
RUN curl -sSL "https://golang.org/dl/go${GOLANG_VERSION}.linux-${ARCH}.tar.gz" -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz

# Install golangci-lint
RUN curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh -o /tmp/install.sh \
    && chmod +x /tmp/install.sh \
    && /tmp/install.sh -b /usr/local/bin ${GOLANGCI_LINT_VERSION}


ENV SRC_BRANCH=${SRC_BRANCH}
ENV SRC_TAG=${SRC_TAG}

RUN git clone https://github.com/longhorn/dep-versions.git -b ${SRC_BRANCH} /usr/src/dep-versions && \
    cd /usr/src/dep-versions && \
    if [ -n "${SRC_TAG}" ] && git show-ref --tags ${SRC_TAG} > /dev/null 2>&1; then \
        echo "Checking out tag ${SRC_TAG}"; \
        cd /usr/src/dep-versions && git checkout tags/${SRC_TAG}; \
    fi

# Build spdk
RUN export REPO_OVERRIDE="" && \
    export COMMIT_ID_OVERRIDE="" && \
    bash /usr/src/dep-versions/scripts/build-spdk.sh "${REPO_OVERRIDE}" "${COMMIT_ID_OVERRIDE}" "${ARCH}"

# Build libjson-c-devel
RUN export REPO_OVERRIDE="" && \
    export COMMIT_ID_OVERRIDE="" && \
    bash /usr/src/dep-versions/scripts/build-libjsonc.sh "${REPO_OVERRIDE}" "${COMMIT_ID_OVERRIDE}"

# Build nvme-cli
RUN export REPO_OVERRIDE="" && \
    export COMMIT_ID_OVERRIDE="" && \
    bash /usr/src/dep-versions/scripts/build-nvme-cli.sh "${REPO_OVERRIDE}" "${COMMIT_ID_OVERRIDE}"

RUN ldconfig

WORKDIR /go/src/github.com/longhorn/go-spdk-helper
COPY . .

FROM base AS validate
RUN ./scripts/validate && touch /validate.done

FROM scratch AS ci-artifacts
COPY --from=validate /validate.done /validate.done
