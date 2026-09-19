#!/usr/bin/env bash
set -euo pipefail

cd "$GITHUB_WORKSPACE"
gocredits_version="v0.4.0"
gocredits_bin="$(mktemp -d "${RUNNER_TEMP%/}/gocredits.XXXXXX")"
previous_credits="$(mktemp "${RUNNER_TEMP%/}/gocredits-credits.XXXXXX")"
trap 'rm -rf "$gocredits_bin" "$previous_credits"' EXIT

sh "$GITHUB_ACTION_PATH/install.sh" \
  -b "$gocredits_bin" "$gocredits_version"
export PATH="$gocredits_bin:$PATH"

module_directory="$(cd "$GOCREDITS_DIRECTORY" && pwd -P)"
credits="$module_directory/CREDITS"
credits_existed=false
if [[ -f "$credits" ]]; then
  cp "$credits" "$previous_credits"
  credits_existed=true
fi

args=(-w)
case "$GOCREDITS_SKIP_MISSING" in
  true)
    args+=(-skip-missing)
    ;;
  false)
    ;;
  *)
    echo "::error::skip-missing must be either true or false"
    exit 1
    ;;
esac
if [[ -n "$GOCREDITS_FORMAT" ]]; then
  args+=(-f "$GOCREDITS_FORMAT")
fi

gocredits "${args[@]}" "$module_directory"

changed=true
if [[ "$credits_existed" == true ]] && cmp -s "$previous_credits" "$credits"; then
  changed=false
fi
{
  echo "credits=$credits"
  echo "changed=$changed"
} >> "$GITHUB_OUTPUT"
