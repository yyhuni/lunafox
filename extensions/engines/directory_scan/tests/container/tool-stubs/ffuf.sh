#!/bin/sh
set -eu

target=
previous=
for argument in "$@"; do
  if [ "$previous" = "-u" ]; then
    target="$argument"
  fi
  previous="$argument"
done

test -n "$target"
if [ "$target" = "http://example.comFUZZ" ]; then
  printf '%s\n' "$@" > /workspace/ffuf.argv
fi

printf '%s\n' '{"input":{"FUZZ":"admin"},"position":1,"status":200,"length":0,"words":0,"lines":0,"content-type":"","redirectlocation":"","scraper":{},"duration":0,"resultfile":"","url":"https://example.com/discovered","host":"example.com"}'
