#!/bin/bash

OS=$(uname|tr '[:upper:]' '[:lower:]')
HARDWARE=arm64
if [ "$(uname -m)" == "x86_64" ]; then
  HARDWARE=amd64
fi

WORKSPACE=${WORKSPACE:-$PWD}

cd $WORKSPACE

# Download ctl
mkdir -p $WORKSPACE/.snz1dp/bin
wget -O $WORKSPACE/.snz1dp/bin/snz1dpctl \
  {{ .DownloadURL }}snz1dpctl-$OS-$HARDWARE
chmod +x $WORKSPACE/.snz1dp/bin/snz1dpctl

export SNZ1DP_HOME=$WORKSPACE/.snz1dp
export PATH=$PATH:$SNZ1DP_HOME/bin

snz1dpctl profile login {{ .ServerURL }} --username "$DPCTL_USERNAME" --password  "$DPCTL_PASSWORD"
snz1dpctl make build
