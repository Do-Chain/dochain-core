# Manual V19 Upgrade: DODEX Wasm Upload Whitelist

Use this only when the validator operators agree to coordinate a manual chain
software upgrade instead of a governance proposal.

## What V19 Changes

At the chosen upgrade height, v19 sets:

- `code_upload_access.permission`: `AnyOfAddresses`
- `code_upload_access.addresses`: includes `do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg`
- `instantiate_default_permission`: `Nobody`

No stores are added, deleted, or renamed.

## Build And Install The Same Binary Everywhere

Build the patched `dochaind` binary from this repo and copy the exact same
binary to every validator/full node that will continue following the chain.

Verify each server reports the new binary after installation:

```bash
dochaind version
```

## Pick One Shared Upgrade Height

On a node with RPC access:

```bash
dochaind status --node http://178.63.79.250:26657
```

Pick a future height far enough away for every server to receive the binary and
plan file. Every node must use the exact same height.

## Put The Manual Plan On Every Node

First find the node home used by the service. Common values are `/root/.do` and
`/home/dochain/.do`.

```bash
systemctl cat dochaind 2>/dev/null | grep -E -- '--home|ExecStart' || true
```

Then create the manual plan in that home directory's `data` folder. Replace
`PUT_HEIGHT_HERE` with the agreed future height.

```bash
DO_HOME=/root/.do
mkdir -p "$DO_HOME/data"

cat > "$DO_HOME/data/manual-v19-upgrade.json" <<'EOF'
{
  "name": "v19",
  "height": PUT_HEIGHT_HERE,
  "info": "Whitelist do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg for CosmWasm uploads and keep default instantiate permission Nobody."
}
EOF
```

If the service uses `/home/dochain/.do`, set:

```bash
DO_HOME=/home/dochain/.do
```

## Restart All Nodes Before The Height

Restart each node after the new binary and plan file are in place:

```bash
systemctl restart dochaind
systemctl status dochaind --no-pager
```

The upgrade handler will run automatically at `PUT_HEIGHT_HERE`.

## Verify After The Height

```bash
dochaind query wasm params \
  --node http://178.63.79.250:26657 \
  --output json
```

Expected:

```json
{
  "code_upload_access": {
    "permission": "AnyOfAddresses",
    "addresses": [
      "do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg"
    ]
  },
  "instantiate_default_permission": "Nobody"
}
```

Then run the DODEX repair scripts from the DEX/deploy server.
