#!/usr/bin/env bash
# This script generates a JSON file containing the mappings of package names to
# team names.
set -euo pipefail

url="http://reqorts.qa.ubuntu.com/m-r-package-team-mapping.json"

curl -fSL -o package_team_mappings.json "$url"
