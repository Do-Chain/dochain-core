#!/bin/bash

set -euo pipefail

# should make this auto fetch upgrade name from app upgrades once many upgrades have been done
# this command will retrieve the folder with the largest number in format v<number>
SOFTWARE_UPGRADE_NAME=$(ls -d -- ./app/upgrades/v* | sort -Vr | head -n 1 | xargs basename)
NODE1_HOME=node1/dochaind
BINARY_OLD="docker exec dochainnode1 ./old/dochaind"
TESTNET_NVAL=${1:-7}

# sleep to wait for localnet to come up
sleep 10

# Schedule the upgrade from the running chain state. Tendermint/CometBFT
# versions differ in whether status JSON uses upper- or lower-case field names.
STATUS_JSON=$($BINARY_OLD status --home "$NODE1_HOME" || true)
CHAIN_ID=$(jq -r '.NodeInfo.network // .node_info.network // empty' <<<"$STATUS_JSON")
CURRENT_HEIGHT=$(jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height // empty' <<<"$STATUS_JSON")
if [[ -z "$CHAIN_ID" || ! "$CURRENT_HEIGHT" =~ ^[0-9]+$ ]]; then
    echo "ERROR: could not read chain ID and block height from node status"
    echo "$STATUS_JSON"
    exit 1
fi
echo "$CHAIN_ID $CURRENT_HEIGHT"
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 20))
echo "$UPGRADE_HEIGHT"

docker exec dochainnode1 tar -cf ./dochaind.tar -C . dochaind
SUM=$(docker exec dochainnode1 sha256sum ./dochaind.tar | cut -d ' ' -f1)
DOCKER_BASE_PATH=$(docker exec dochainnode1 pwd)
echo $SUM
UPGRADE_INFO=$(jq -n '
{
    "binaries": {
        "linux/amd64": "file://'$DOCKER_BASE_PATH'/dochaind.tar?checksum=sha256:'"$SUM"'",
    }
}')

# The upgrade network mirrors mainnet governance: each validator stakes DODX,
# while transaction fees continue to be paid in DO.
for (( i=0; i<TESTNET_NVAL; i++ )); do
    if [[ $(docker ps -a | grep dochainnode$i | wc -l) -eq 1 ]]; then
        $BINARY_OLD tx dodxstaking stake "10000000udodx" --from node$i --keyring-backend test --chain-id "$CHAIN_ID" --home "node$i/dochaind" --fees "1000udo" -y
        sleep 2
    fi
done

GOV_AUTHORITY=$($BINARY_OLD q auth module-account gov --home "$NODE1_HOME" --output json | jq -r '.account.value.address // .account.base_account.address // .account.address')
if [[ -z "$GOV_AUTHORITY" || "$GOV_AUTHORITY" == "null" ]]; then
    echo "ERROR: could not resolve the governance module authority"
    exit 1
fi

PROPOSAL_FILE=$(mktemp)
trap 'rm -f "$PROPOSAL_FILE"' EXIT
jq -n \
    --arg authority "$GOV_AUTHORITY" \
    --arg name "$SOFTWARE_UPGRADE_NAME" \
    --arg height "$UPGRADE_HEIGHT" \
    --arg info "$UPGRADE_INFO" \
    '{
        messages: [{
            "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
            authority: $authority,
            plan: {name: $name, height: $height, info: $info}
        }],
        deposit: "20000000udodx",
        title: ("upgrade to " + $name),
        summary: ("upgrade to " + $name),
        metadata: "",
        expedited: false
    }' > "$PROPOSAL_FILE"

$BINARY_OLD tx gov submit-proposal "$PROPOSAL_FILE" --from node1 --keyring-backend test --chain-id "$CHAIN_ID" --home "$NODE1_HOME" --fees "1000udo" -y

sleep 5

# loop from 0 to TESTNET_NVAL
for (( i=0; i<$TESTNET_NVAL; i++ )); do
    # check if docker for node i is running
    if [[ $(docker ps -a | grep dochainnode$i | wc -l) -eq 1 ]]; then
        $BINARY_OLD tx gov vote 1 yes --from node$i --keyring-backend test --chain-id "$CHAIN_ID" --home "node$i/dochaind" --fees "1000udo" -y
        sleep 5
    fi
done

# keep track of block_height
NIL_BLOCK=0
LAST_BLOCK=0
SAME_BLOCK=0
while true; do 
    STATUS_JSON=$($BINARY_OLD status --home "$NODE1_HOME" || true)
    BLOCK_HEIGHT=$(jq -r '.SyncInfo.latest_block_height // .sync_info.latest_block_height // empty' <<<"$STATUS_JSON")
    # Treat missing and malformed heights as transient status failures.
    if [[ ! $BLOCK_HEIGHT =~ ^[0-9]+$ ]]; then
        # if 5 nil blocks in a row, exit
        if [[ $NIL_BLOCK -ge 5 ]]; then
            echo "ERROR: node status returned no valid block height 5 times in a row"
            docker logs dochainnode0
            exit 1
        fi
        NIL_BLOCK=$((NIL_BLOCK + 1))
        sleep 10
        continue
    fi
    NIL_BLOCK=0

    # if block height is not nil
    # if block height is same as last block height
    if [[ $BLOCK_HEIGHT -eq $LAST_BLOCK ]]; then
        # if 5 same blocks in a row, exit
        if [[ $SAME_BLOCK -ge 5 ]]; then
            echo "ERROR: 5 same blocks in a row"
            break
        fi
        SAME_BLOCK=$((SAME_BLOCK + 1))
    else
        # update LAST_BLOCK and reset SAME_BLOCK
        LAST_BLOCK=$BLOCK_HEIGHT
        SAME_BLOCK=0
    fi

    if [[ $BLOCK_HEIGHT -ge $UPGRADE_HEIGHT ]]; then
        # assuming running only 1 dochaind
        echo "UPGRADE REACHED, CONTINUING NEW CHAIN"
        break
    else
        $BINARY_OLD q gov proposal 1 --output=json --home "$NODE1_HOME" | jq ".status"
        echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
        sleep 10
    fi
done

if [[ $SAME_BLOCK -ge 5 ]]; then
    docker logs dochainnode0
    exit 1
fi

sleep 40

# check all nodes are online after upgrade
for (( i=0; i<$TESTNET_NVAL; i++ )); do
    if [[ $(docker ps -a | grep dochainnode$i | wc -l) -eq 1 ]]; then
        docker exec dochainnode$i ./dochaind status --home "node$i/dochaind"
        if [[ "${PIPESTATUS[0]}" != "0" ]]; then
            echo "node$i is not online"
            docker logs dochainnode$i
            exit 1
        fi
    else
        echo "dochainnode$i is not running"
        docker logs dochainnode$i
        exit 1
    fi
done






