#!/usr/bin/env bash

if [ "${NO_COLOR:-}" = "1" ]; then
  RED=""
  GREEN=""
  YELLOW=""
  CYAN=""
  RESET=""
else
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[0;33m'
  CYAN='\033[0;36m'
  RESET='\033[0m'
fi

info() {
  echo -e "${CYAN}[INFO]${RESET} $*"
}

warn() {
  echo -e "${YELLOW}[WARN]${RESET} $*"
}

error() {
  echo -e "${RED}[ERROR]${RESET} $*" >&2
}

success() {
  echo -e "${GREEN}[OK]${RESET} $*"
}

usage_error() {
  local message="$1"
  error "$message"
  if declare -f usage >/dev/null 2>&1; then
    usage >&2
  fi
  exit 2
}

normalize_exit_codes() {
  trap '_normalize_exit_codes_handler' EXIT
}

_normalize_exit_codes_handler() {
  local rc=$?
  trap - EXIT
  case "$rc" in
    0|2)
      exit "$rc"
      ;;
    *)
      exit 1
      ;;
  esac
}
