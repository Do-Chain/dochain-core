#!/usr/bin/env bash

set -euo pipefail

# this bash will prepare cosmosvisor to the build folder so that it can perform upgrade
# this script is supposed to be run by Makefile

# Build the exact predecessor currently running on Do-Chain mainnet. Pinning the
# commit and archive digest prevents a moved tag or modified download from
# silently changing the upgrade test's starting binary.
OLD_COMMIT=d9139025424184098dafb1fd67ef74499cf2e0b6
OLD_ARCHIVE_SHA256=ade8b21a2cdc5c4eea12d427bd758eaf670bfd40079909a0f451e3094af59334
OLD_SOURCE_DIR=dochain-core-${OLD_COMMIT}
OLD_BASE_IMAGE=golang:1.24.7-alpine3.22
# this command will retrieve the folder with the largest number in format v<number>
SOFTWARE_UPGRADE_NAME=$(ls -d -- ./app/upgrades/v* | sort -Vr | head -n 1 | xargs basename)
BUILDDIR=${1:-}
TESTNET_NVAL=${2:-}
TESTNET_CHAINID=${3:-}

# check if BUILDDIR is set
if [ -z "$BUILDDIR" ]; then
    echo "BUILDDIR is not set"
    exit 1
fi
if [ -z "$TESTNET_NVAL" ] || [ -z "$TESTNET_CHAINID" ]; then
    echo "TESTNET_NVAL and TESTNET_CHAINID must be set"
    exit 1
fi

# install old version of dochaind

## Fetch and verify the exact live predecessor source.
if [ ! -d "_build/${OLD_SOURCE_DIR}" ]; then
    mkdir -p _build
    OLD_ARCHIVE="_build/${OLD_COMMIT}.tar.gz"
    curl --fail --location --retry 3 \
        "https://codeload.github.com/Do-Chain/dochain-core/tar.gz/${OLD_COMMIT}" \
        --output "${OLD_ARCHIVE}"
    echo "${OLD_ARCHIVE_SHA256}  ${OLD_ARCHIVE}" | sha256sum --check --strict
    tar -xzf "${OLD_ARCHIVE}" -C _build
fi
test -f "_build/${OLD_SOURCE_DIR}/go.mod"

## check if $BUILDDIR/old/dochaind exists
if [ ! -f "$BUILDDIR/old/dochaind" ]; then
    mkdir -p "$BUILDDIR/old"
    docker build --platform linux/amd64 --no-cache \
        --build-arg "source=./_build/${OLD_SOURCE_DIR}/" \
        --build-arg "BASE_IMAGE=${OLD_BASE_IMAGE}" \
        --build-arg "GIT_COMMIT=${OLD_COMMIT}" \
        --build-arg "GIT_VERSION=v1.0.1-rc6" \
        --tag dochain/dochaind-binary.old \
        -f contrib/updates/Dockerfile.old .
    docker create --platform linux/amd64 --name old-temp dochain/dochaind-binary.old:latest
    docker cp old-temp:/usr/local/bin/dochaind "$BUILDDIR/old/"
    docker rm old-temp
fi

# prepare cosmovisor config in TESTNET_NVAL nodes
if [ ! -f "$BUILDDIR/node0/dochaind/config/genesis.json" ]; then docker run --rm \
    --user $(id -u):$(id -g) \
    -v "$BUILDDIR:/dochaind:Z" \
    -v /etc/group:/etc/group:ro \
    -v /etc/passwd:/etc/passwd:ro \
    -v /etc/shadow:/etc/shadow:ro \
    --entrypoint /dochaind/old/dochaind \
    --platform linux/amd64 \
    daviddochain/dochaind-upgrade-env testnet --v "$TESTNET_NVAL" --chain-id "$TESTNET_CHAINID" -o . --starting-ip-address 192.168.10.2 --keyring-backend=test --home=temp; \
fi

for (( i=0; i<$TESTNET_NVAL; i++ )); do
    CURRENT=$BUILDDIR/node$i/dochaind

    # Model mainnet's token roles in the upgrade network: DODX supplies
    # governance voting power while DO remains the validator bond and fee denom.
    DODX_PER_VALIDATOR=100000000
    DODX_TOTAL=$((DODX_PER_VALIDATOR * TESTNET_NVAL))
    jq \
        --arg dodx_per_validator "$DODX_PER_VALIDATOR" \
        --arg dodx_total "$DODX_TOTAL" \
        '
        .app_state.gov.voting_params.voting_period = "50s"
        | .app_state.bank.balances |= map(
            .coins = ((.coins + [{"denom": "udodx", "amount": $dodx_per_validator}]) | sort_by(.denom))
        )
        | .app_state.bank.supply = ((
            .app_state.bank.supply + [{"denom": "udodx", "amount": $dodx_total}]
          ) | sort_by(.denom))
        | .app_state.dodxstaking.governance_enabled = true
        ' "$CURRENT/config/genesis.json" > "$CURRENT/config/genesis.json.tmp"
    mv "$CURRENT/config/genesis.json.tmp" "$CURRENT/config/genesis.json"

    docker run --rm \
        --user $(id -u):$(id -g) \
        -v "$BUILDDIR:/dochaind:Z" \
        -v /etc/group:/etc/group:ro \
        -v /etc/passwd:/etc/passwd:ro \
        -v /etc/shadow:/etc/shadow:ro \
        -e DAEMON_HOME=/dochaind/node$i/dochaind \
        -e DAEMON_NAME=dochaind \
        -e DAEMON_RESTART_AFTER_UPGRADE=true \
        --entrypoint /dochaind/cosmovisor \
        --platform linux/amd64 \
        daviddochain/dochaind-upgrade-env init /dochaind/old/dochaind
    mkdir -p "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin"
    cp "$BUILDDIR/dochaind" "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin"
    touch "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/upgrade-info.json"
done





