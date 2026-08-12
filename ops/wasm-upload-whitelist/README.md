# DODEX Wasm Upload Whitelist

This folder contains the governance payload to whitelist the controlled DODEX
admin wallet for CosmWasm uploads without opening uploads to everybody.

## Target State

- `code_upload_access`: `AnyOfAddresses`
- allowed upload address: `do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg`
- `instantiate_default_permission`: `Nobody`
- wasm authority: `do10d07y265gmmuvt4z0w9aw880jnsr700jslsmp2`

## Submit The Proposal

Use the full params proposal when current upload access is `Nobody`.

```bash
dochaind tx gov submit-proposal \
  ops/wasm-upload-whitelist/wasm-upload-whitelist-do1t7-proposal.json \
  --from foundation \
  --keyring-backend test \
  --chain-id Do-Chain \
  --node http://178.63.79.250:26657 \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees 50000udo \
  --yes
```

From the server copy in `/root/do-predict-deploy`, use:

```bash
cd /root/do-predict-deploy

dochaind tx gov submit-proposal \
  wasm-upload-whitelist/wasm-upload-whitelist-do1t7-proposal.json \
  --from foundation \
  --keyring-backend test \
  --chain-id Do-Chain \
  --node http://178.63.79.250:26657 \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees 50000udo \
  --yes
```

If the live chain is already using `AnyOfAddresses` and only needs this address
added, the narrower wasm proposal command can be used instead:

```bash
dochaind tx wasm submit-proposal add-code-upload-params-addresses \
  do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg \
  --authority do10d07y265gmmuvt4z0w9aw880jnsr700jslsmp2 \
  --title "Whitelist DODEX factory admin for CosmWasm uploads" \
  --summary "Add do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg to x/wasm code upload allowlist." \
  --deposit 1000000udodx \
  --from foundation \
  --keyring-backend test \
  --chain-id Do-Chain \
  --node http://178.63.79.250:26657 \
  --gas auto \
  --gas-adjustment 1.3 \
  --fees 50000udo \
  --yes
```

The narrow command will fail if `code_upload_access.permission` is still
`Nobody`; in that case use the full params proposal JSON above.

## Verify After Passing

```bash
dochaind query wasm params \
  --node http://178.63.79.250:26657 \
  --output json
```

Expected values:

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

## Run DODEX Upload Repair After Whitelist

```bash
cd /root/do-predict-deploy

export LP_WASM=/root/cw20_base_v1.1.2.wasm
export EXPECTED_UPLOADER=do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg

read -s DEPLOYER_MNEMONIC
export DEPLOYER_MNEMONIC

node reupload-unlock.mjs
```

Then use the printed new code IDs:

```bash
export NEW_PAIR_XYK_CODE_ID=PUT_NEW_PAIR_CODE_ID_HERE
export NEW_LP_TOKEN_CODE_ID=PUT_NEW_LP_CODE_ID_HERE
export ADMIN_MNEMONIC="$DEPLOYER_MNEMONIC"
export EXPECTED_ADMIN=do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg

node fix-factory-codeids.mjs
```
