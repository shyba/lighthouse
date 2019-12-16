#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
(
  touch -a .env && set -o allexport
  source ./.env; set +o allexport
  hash reflex 2>/dev/null || go get github.com/cespare/reflex
  hash reflex 2>/dev/null || { echo >&2 'Make sure '"$(go env GOPATH)"'/bin is in your $PATH'; exit 1;  }

  reflex --decoration=none --start-service=true --regex='\.go$' -- sh -c "go generate && go run *.go serve -d"
)
