#!/bin/sh

YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

warn() {
  printf '%s\n' "${YELLOW}! $*${NO_COLOR}"
}

check_bin_dir() {
  bin_dir="$1"

  # https://stackoverflow.com/a/11655875
  good=$(
    IFS=:
    for path in $PATH; do
      if [ "${path}" = "${bin_dir}" ]; then
        printf 1
        break
      fi
    done
  )

  if [ "${good}" != "1" ]; then
    warn "Bin directory ${bin_dir} is not in your \$PATH"
  fi
}

install() {
  case "$(uname -s)" in
  Linux)
    PLATFORM='linux'
    ;;
  *)
    echo "Platform $(uname -s) may not be supported."
    exit 1
    ;;
  esac

  case "$(uname -m)" in
  x86_64)
    ARCH="x86_64"
    ;;
  armv5*)
    ARCH="armv5"
    ;;
  armv6*)
    ARCH="armv6"
    ;;
  armv7*)
    ARCH="armv7"
    ;;
  arm)
    ARCH="armv6"
    ;;
  armv8*)
    ARCH="arm64"
    ;;
  arm64)
    ARCH="arm64"
    ;;
  aarch64*)
    ARCH="arm64"
    ;;
  *)
    echo "Architect $(uname -m) may not be supported."
    exit 1
    ;;
  esac

  set -e

  temp_file=$(mktemp /tmp/gg.XXXXXXXXX)
  curl -L "https://github.com/mzz2017/gg/releases/latest/download/gg-${PLATFORM}-${ARCH}" -o "${temp_file}"
  chmod +x "${temp_file}"
  setcap cap_net_raw+ep "${temp_file}" >/dev/null 2>&1 || true
  if [ -w /usr/local/bin/gg ]; then
    bin_dir=/usr/local/bin/gg
  else
    bin_dir="${HOME}"/.local/bin/gg
  fi
  check_bin_dir "${bin_dir}"
}

install
gg --version
