#!/bin/bash

set -e -o pipefail

function main() {
  local -r number="$1"

  for (( i=1; i<=$number; i++ )); do
    echo "Creating $i consumers"
    STREAM="$STREAM-$i" GROUP="$i" nohup make consumer > logs/consumer-$i.log 2>&1 &
  done

  echo "Watch the logs with: tail -f logs/consumer-*.log"
  wait < <(jobs -p)
}

main "$@"