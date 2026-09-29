#!/usr/bin/env bash
set -euo pipefail

cd "$GITHUB_WORKSPACE"
gocredits_version="${GOCREDITS_TEST_VERSION:-v1.0.2}"
gocredits_bin="${RUNNER_TEMP%/}/gocredits/bin"

case "$GOCREDITS_RUN" in
  true | false)
    ;;
  *)
    echo "::error::run must be either true or false"
    exit 1
    ;;
esac

sh "$GITHUB_ACTION_PATH/install.sh" \
  -b "$gocredits_bin" "$gocredits_version"
echo "$gocredits_bin" >> "$GITHUB_PATH"

if [[ "$GOCREDITS_RUN" == false ]]; then
  exit 0
fi

gocredits_executable="$gocredits_bin/gocredits"
if [[ "${RUNNER_OS:-}" == Windows ]]; then
  gocredits_executable="${gocredits_executable}.exe"
fi

previous_credits="$(mktemp "${RUNNER_TEMP%/}/gocredits-credits.XXXXXX")"
trap 'rm -f "$previous_credits"' EXIT

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

"$gocredits_executable" "${args[@]}" "$module_directory"

changed=true
if [[ "$credits_existed" == true ]] && cmp -s "$previous_credits" "$credits"; then
  changed=false
fi
credits_output="$credits"
if [[ "${RUNNER_OS:-}" == Windows ]]; then
  credits_output="$(cygpath -m "$credits")"
fi
{
  echo "credits=$credits_output"
  echo "changed=$changed"
} >> "$GITHUB_OUTPUT"
