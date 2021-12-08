#!/bin/bash

case "$(uname -s)" in
 Linux)
   PLATFORM='linux'
   ;;

 *)
   echo "Platform $(uname -s) may not be supported."
   exit 1
   ;;
esac

if [[ "$(uname -m)" == 'x86_64' ]]; then
  ARCH="x86_64"
elif [[ "$(uname -m)" == armv5* ]]; then
  ARCH="armv5"
elif [[ "$(uname -m)" == armv6* ]]; then
  ARCH="armv6"
elif [[ "$(uname -m)" == armv7* ]]; then
  ARCH="armv7"
elif [[ "$(uname -m)" == 'arm' ]]; then
  ARCH="armv6"
elif [[ "$(uname -m)" == 'arm64' ]]; then
  ARCH="arm64"
elif [[ "$(uname -m)" == 'aarch64' ]]; then
  ARCH="arm64"
else
  echo "Architect $(uname -m) may not be supported."
  exit 1
fi

curl -L "https://github.com/mzz2017/gg/releases/latest/download/gg-${PLATFORM}-${ARCH}" -o /usr/local/bin/gg
chmod +x /usr/local/bin/gg
gg --version
