#!/usr/bin/env bash
set -e

DEFAULT_BIN_DIR="/usr/local/bin"
BIN_DIR=${1:-"${DEFAULT_BIN_DIR}"}
GITHUB_REPO="kaytu-io/pennywise"
TMP_DIR=$(mktemp -d -t pennywise-install.XXXXXXXXXX)

# Helper functions for logs
info() {
    echo '[INFO] ' "$@"
}

warn() {
    echo '[WARN] ' "$@" >&2
}

fatal() {
    echo '[ERROR] ' "$@" >&2
    exit 1
}

# Set os, fatal if operating system not supported
setup_verify_os() {
    OS=$(uname -s)
    case ${OS} in
        Darwin)
            OS=darwin
            ;;
        Linux)
            OS=linux
            ;;
        *)
            fatal "Unsupported operating system ${OS}"
    esac
}

# Set arch, fatal if architecture not supported
setup_verify_arch() {
    ARCH=$(uname -m)

    case ${ARCH} in
        arm64|aarch64|armv8l)
            ARCH=arm64
            ;;
        amd64)
            ARCH=amd64
            ;;
        x86_64)
            ARCH=amd64
            ;;
        *)
            fatal "Unsupported architecture ${ARCH}"
    esac
}

# Verify existence of downloader executable
verify_downloader() {
    # Return failure if it doesn't exist or is no executable
    [ -x "$(which "$1")" ] || return 1

    # Set verified executable as our downloader program and return success
    DOWNLOADER=$1
    return 0
}


# Find version from Github metadata
get_release_version() {
    if [ -n "${FLUX_VERSION}" ]; then
      SUFFIX_URL="tags/v${FLUX_VERSION}"
    else
      SUFFIX_URL="latest"
    fi

    METADATA_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/${SUFFIX_URL}"
    TMP_METADATA="${TMP_DIR}/pennywise.json"
    info "Downloading metadata ${METADATA_URL}"
    download "${TMP_METADATA}" "${METADATA_URL}"
    VERSION=$(grep '"tag_name":' "${TMP_METADATA}" | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)
    if [ -n "${VERSION}" ]; then
        info "Using ${VERSION} as release"
    else
        fatal "Unable to determine release version"
    fi
}

# Download from file from URL
download() {
    [ $# -eq 2 ] || fatal 'download needs exactly 2 arguments'

    case $DOWNLOADER in
        curl)
            curl -o "$1" -sfL "$2"
            ;;
        wget)
            wget --auth-no-challenge -qO "$1" "$2"
            ;;
        *)
            fatal "Incorrect executable '${DOWNLOADER}'"
            ;;
    esac

    # Abort if download command failed
    [ $? -eq 0 ] || fatal 'Download failed'
}

# Download hash from Github URL
download_hash() {
    HASH_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/pennywise_${VERSION}_checksums.txt"
    TMP_HASH="${TMP_DIR}/pennywise.hash"
    info "Downloading hash ${HASH_URL}"
    download "${TMP_HASH}" "${HASH_URL}"
    HASH_EXPECTED=$(grep " pennywise_${VERSION}_${OS}_${ARCH}.tar.gz$" "${TMP_HASH}")
    HASH_EXPECTED=${HASH_EXPECTED%%[[:blank:]]*}
}


# Download binary from Github URL
download_binary() {
    BIN_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/pennywise_${VERSION}_${OS}_${ARCH}.tar.gz"
    info "Downloading binary ${BIN_URL}"
    TMP_BIN="${TMP_DIR}/pennywise.tar.gz"

    download "${TMP_BIN}" "${BIN_URL}"
}

compute_sha256sum() {
  cmd=$(which sha256sum shasum | head -n 1)
  case $(basename "$cmd") in
    sha256sum)
      sha256sum "$1" | cut -f 1 -d ' '
      ;;
    shasum)
      shasum -a 256 "$1" | cut -f 1 -d ' '
      ;;
    *)
      fatal "Can not find sha256sum or shasum to compute checksum"
      ;;
  esac
}

# Verify downloaded binary hash
verify_binary() {
    info "Verifying binary download"
    HASH_BIN=$(compute_sha256sum "${TMP_BIN}")
    HASH_BIN=${HASH_BIN%%[[:blank:]]*}
    if [ "${HASH_EXPECTED}" != "${HASH_BIN}" ]; then
        fatal "Download sha256 does not match ${HASH_EXPECTED}, got ${HASH_BIN}"
    fi
}

# Setup permissions and move binary
setup_binary() {
    chmod 755 "${TMP_BIN}"
    info "Installing pennywise to ${BIN_DIR}/pennywise"
    tar -xzof "${TMP_BIN}" -C "${TMP_DIR}"

    local CMD_MOVE="mv -f \"${TMP_DIR}/pennywise\" \"${BIN_DIR}\""
    if [ -w "${BIN_DIR}" ]; then
        eval "${CMD_MOVE}"
    else
        eval "sudo ${CMD_MOVE}"
    fi
    info "Installed successfully"
}

# Run the install process
{
    setup_verify_os
    setup_verify_arch
    verify_downloader curl || verify_downloader wget || fatal 'Can not find curl or wget for downloading files'
    get_release_version
    download_hash
    download_binary
    verify_binary
    setup_binary
}