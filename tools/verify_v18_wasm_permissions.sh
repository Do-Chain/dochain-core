#!/usr/bin/env bash

set -euo pipefail

DOCHAIND=${DOCHAIND:-dochaind}
NODE_ARGS=${NODE_ARGS:-}

params_json=$($DOCHAIND query wasm params -o json $NODE_ARGS)
upload_permission=$(jq -r '.params.code_upload_access.permission // .code_upload_access.permission // empty' <<<"$params_json")
instantiate_permission=$(jq -r '.params.instantiate_default_permission // .instantiate_default_permission // empty' <<<"$params_json")

if [ "$upload_permission" != "Nobody" ] && [ "$upload_permission" != "ACCESS_TYPE_NOBODY" ]; then
  echo "unexpected code upload permission: ${upload_permission:-missing}" >&2
  exit 1
fi

if [ "$instantiate_permission" != "Nobody" ] && [ "$instantiate_permission" != "ACCESS_TYPE_NOBODY" ]; then
  echo "unexpected instantiate default permission: ${instantiate_permission:-missing}" >&2
  exit 1
fi

codes_json=$($DOCHAIND query wasm list-code -o json $NODE_ARGS)
open_codes=$(jq -r '
  [.code_infos[]? | select(
    (.instantiate_permission.permission // .instantiate_config.permission // "") as $p |
    $p != "" and $p != "Nobody" and $p != "ACCESS_TYPE_NOBODY"
  ) | (.code_id // .codeID // .id)] | join(",")
' <<<"$codes_json")

if [ -n "$open_codes" ]; then
  echo "code IDs still allow instantiate: $open_codes" >&2
  exit 1
fi

echo "v18 wasm permissions verified"
