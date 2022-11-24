#!/bin/bash

set -e -o pipefail

function main() {
  local -r number="$1"

  for (( i=1; i<=$number; i++ )); do
    echo "Creating $i producer"
    STREAM="$STREAM-$i" nohup make producer > logs/producer-$i.log 2>&1 &
  done

  echo "Watch the logs with: tail -f logs/producer-*.log"
  wait < <(jobs -p)
}

main "$@"