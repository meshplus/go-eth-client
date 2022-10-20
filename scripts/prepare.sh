#!/usr/bin/env bash

BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function go_install() {
  version=$(go env GOVERSION)
  if [[ ! "$version" < "go1.16" ]];then
      go install "$@"
  else
      go get "$@"
  fi
}

print_blue "===> 1. Install pip3"
if ! type pip3 >/dev/null 2>&1; then
  sudo apt install python3-pip
fi

print_blue "===> 2. Install solc"
if ! type solc >/dev/null 2>&1; then
  pip3 uninstall solc-select
  pip3 install solc-select==0.2.0
  solc-select install 0.8.15
  solc-select use 0.8.15
  solc --version
fi





