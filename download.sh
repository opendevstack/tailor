#!/usr/bin/env bash

# Set version to latest unless set by user
if [ -z "$VERSION" ]; then
  VERSION="0.6.1"
fi

echo "Downloading version ${VERSION}..."

# OS information (contains e.g. darwin x86_64)
UNAME=`uname -a | awk '{print tolower($0)}'`
if [[ ($UNAME == *"mac os x"*) || ($UNAME == *darwin*) ]]; then
  PLATFORM="darwin"
  EXT=""
elif [[ ($UNAME == *"NT"*) ]]; then
  PLATFORM="windows"
  EXT=".exe"
else
  PLATFORM="linux"
  EXT=""
fi
if [[ ($UNAME == *x86_64*) || ($UNAME == *amd64*) ]]
then
  ARCH="amd64"
else
  echo "Currently, there are no 32bit binaries provided."
  echo "You will need to build binaries yourself."
  exit 1
fi

# Download binary
echo "Downloading https://github.com/opendevstack/tailor/releases/download/v${VERSION}/tailor_${PLATFORM}_${ARCH}${EXT}"
curl -L -o tailor${EXT} "https://github.com/opendevstack/tailor/releases/download/v${VERSION}/tailor_${PLATFORM}_${ARCH}${EXT}"

# Make binary executable
chmod +x tailor${EXT}

echo "Done."
