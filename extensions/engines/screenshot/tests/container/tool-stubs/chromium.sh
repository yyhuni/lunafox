#!/bin/sh
set -eu
printf '%s\n' "$*" > /workspace/chromium.argv
printf '%s\n' 'Chromium 131.0.6778.85'
exit 0
