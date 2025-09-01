#!/bin/bash

WORKSPACE=${WORKSPACE:-$PWD}
cd $WORKSPACE
export SNZ1DP_HOME=$WORKSPACE/.snz1dp
export PATH=$PATH:$SNZ1DP_HOME/bin

snz1dpctl make publish
