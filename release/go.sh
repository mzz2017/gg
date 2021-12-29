#!/bin/sh

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

set -ex

curl -L "https://hubmirror.v2raya.org/mzz2017/gg/releases/latest/download/gg-${PLATFORM}-${ARCH}" -o /usr/local/bin/gg
chmod +x /usr/local/bin/gg
setcap cap_net_raw+ep  /usr/local/bin/gg
gg --version
