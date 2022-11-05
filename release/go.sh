#!/bin/sh

YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

warn() {
  printf '%s\n' "${YELLOW}! $*${NO_COLOR}"
}

_warn() {
  printf '%s\n' "${YELLOW}$*${NO_COLOR}"
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

download_and_install() {
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
  trap "rm -f '$temp_file'" exit
  curl -L "https://github.com/mzz2017/gg/releases/latest/download/gg-${PLATFORM}-${ARCH}" -o "${temp_file}"
  all_user_access=0
  touch /usr/local/bin/gg > /dev/null 2>&1 && all_user_access=1
  if [ "$all_user_access" = 1 ]; then
    bin_dir=/usr/local/bin
  else
    bin_dir="${HOME}"/.local/bin
  fi
  check_bin_dir "${bin_dir}"
  install -vDm755 "${temp_file}" "${bin_dir}/gg"
  cap_set=0
  setcap cap_net_raw,cap_sys_ptrace+ep "${bin_dir}/gg" && cap_set=1
  chown root "${bin_dir}/gg" >/dev/null 2>&1 || true
  chgrp root "${bin_dir}/gg" >/dev/null 2>&1 || true
  ptrace_scope=$(cat /proc/sys/kernel/yama/ptrace_scope)
  if [ "$ptrace_scope" = 3 ]; then
    warn "Your kernel does not allow ptrace permission; please use following command and reboot:"
    echo "echo kernel.yama.ptrace_scope = 1 | sudo tee -a /etc/sysctl.d/10-ptrace.conf"
  elif [ "$ptrace_scope" = 2 ] && [ "$cap_set" = 0 ]; then
    warn "Your ptrace_scope is 2 and you should give the correct capability to gg:"
    echo "sudo setcap cap_net_raw,cap_sys_ptrace+ep ""${bin_dir}""/gg"
  fi
}

check_command() {
  echo "$SHELL" | grep "/fish" >/dev/null
  if [ $? = 0 ]; then
    alias_output=$(fish -ic "functions gg --no-details")
  else
    alias_output=$("$SHELL" -ic "command -v gg"); echo "$alias_output" | grep "alias"
  fi
  if [ $? = 0 ]; then
    warn "[Warn] gg conflicts with:"
    echo "$alias_output" | sed "s/^/  /g"
    _warn "please use follolwing commands to resolve it:"
    echo $SHELL | grep "/zsh" >/dev/null && echo '  echo "unalias gg" | tee -a ~/.zshrc; source ~/.zshrc'
    echo $SHELL | grep "/bash" >/dev/null && echo '  echo "unalias gg" | tee -a ~/.bashrc; source ~/.bashrc'
    echo $SHELL | grep "/fish" >/dev/null && echo '  echo "functions --erase gg" | tee -a ~/.config/fish/conf.d/99-go-graft.fish; source ~/.config/fish/conf.d/99-go-graft.fish'
  else
    gg --version
  fi
}

download_and_install
check_command

