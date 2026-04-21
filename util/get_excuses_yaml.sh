#!/usr/bin/env bash
# Fetch the Ubuntu excuses YAML file from the proposed migration path.
set -euo pipefail

url="https://ubuntu-archive-team.ubuntu.com/proposed-migration/update_excuses.yaml.xz"

curl -L -o update_excuses.yaml.xz "$url"
unxz -k update_excuses.yaml.xz
rm update_excuses.yaml.xz
