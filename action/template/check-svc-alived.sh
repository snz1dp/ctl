#!/bin/bash

nc -v -w 2 -z {{ .VRRPCheck.IP }} {{ .VRRPCheck.Port }} 2>&1 | grep open | grep {{ .VRRPCheck.Port }} >/dev/null 2>&1
exit $?
