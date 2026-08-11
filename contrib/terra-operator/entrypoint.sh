#!/bin/sh

# Default to "data".
DATADIR="${DATADIR:-/do/.do/data}"
MONIKER="${MONIKER:-docker-node}"
ENABLE_LCD="${ENABLE_LCD:-true}"
MINIMUM_GAS_PRICES=${MINIMUM_GAS_PRICES-10000000udo}
SNAPSHOT_NAME="${SNAPSHOT_NAME}"
SNAPSHOT_BASE_URL="${SNAPSHOT_BASE_URL:-https://getsfo.quicksync.io}"
PUBLIC_RPC="${PUBLIC_RPC:-false}"
ALLOW_REMOTE_NETWORK_CONFIG="${ALLOW_REMOTE_NETWORK_CONFIG:-false}"

# Moniker will be updated by entrypoint.
dochaind init --chain-id $CHAINID moniker

# Backup for templating
mv ~/.do/config/config.toml ~/config.toml
mv ~/.do/config/app.toml ~/app.toml

download_network_config() {
  if [ "$ALLOW_REMOTE_NETWORK_CONFIG" != "true" ] ; then
    echo "Remote genesis/addrbook downloads are disabled. Set ALLOW_REMOTE_NETWORK_CONFIG=true only after verifying the source." >&2
    exit 1
  fi

  wget -O "$1" "$2"
}

if [ "$CHAINID" = "cookie-1" ] ; then download_network_config ~/.do/config/genesis.json https://columbus-genesis.s3.ap-northeast-1.amazonaws.com/cookie-1-genesis.json; fi
if [ "$CHAINID" = "cookie-1" ] ; then download_network_config ~/.do/config/addrbook.json https://networks.mcontrol.ml/columbus/addrbook.json; fi
if [ "$CHAINID" = "rebel-1" ] ; then download_network_config ~/.do/config/genesis.json https://raw.githubusercontent.com/do-rebels/classic-testnet/master/rebel-1/genesis.json; fi
if [ "$CHAINID" = "rebel-1" ] ; then download_network_config ~/.do/config/addrbook.json https://raw.githubusercontent.com/do-rebels/classic-testnet/master/rebel-1/addrbook.json; fi
if [ "$CHAINID" = "rebel-2" ] ; then download_network_config ~/.do/config/genesis.json https://raw.githubusercontent.com/do-rebels/classic-testnet/master/rebel-2/genesis.json; fi
if [ "$CHAINID" = "rebel-2" ] ; then download_network_config ~/.do/config/addrbook.json https://raw.githubusercontent.com/do-rebels/classic-testnet/master/rebel-2/addrbook.json; fi

# First sed gets the app.toml moved into place.
# app.toml updates
sed 's/minimum-gas-prices = "0udo"/minimum-gas-prices = "'"$MINIMUM_GAS_PRICES"'"/g' ~/app.toml > ~/.do/config/app.toml

# Needed to use awk to replace this multiline string.
if [ "$ENABLE_LCD" = true ] ; then
  sed -i '0,/enable = false/s//enable = true/' ~/.do/config/app.toml

fi

# config.toml updates

sed 's/moniker = "moniker"/moniker = "'"$MONIKER"'"/g' ~/config.toml > ~/.do/config/config.toml
if [ "$PUBLIC_RPC" = "true" ] ; then
  sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.do/config/config.toml
fi

if [ "$CHAINID" = "cookie-1" ] && [ -n "$SNAPSHOT_NAME" ] ; then
  # Download the snapshot if data directory is empty.
  res=$(find "$DATADIR" -name "*.db")
  if [ "$res" ]; then
      echo "data directory is NOT empty, skipping quicksync"
  else
      echo "starting snapshot download"
      mkdir -p $DATADIR
      cd $DATADIR
      FILENAME="$SNAPSHOT_NAME"

      # Download
      aria2c -x5 $SNAPSHOT_BASE_URL/$FILENAME
      # Extract
      lz4 -d $FILENAME | tar xf -

      # # cleanup
      rm $FILENAME
  fi
fi

# check if CHAINID is test
if [ "$NEW_NETWORK" = "true" ] ; then
  # add new gentx
  sh /test-node-setup.sh
fi

dochaind start $DOCHAIND_STARTUP_PARAMETERS &

if [ "$NEW_NETWORK" = "false" ] ; then
  #Wait for dochaind to catch up
  while true
  do
    if ! (( $(echo $(dochaind status) | awk -F '"catching_up":|},"ValidatorInfo"' '{print $2}') ));
    then
      break
    fi
    sleep 1
  done

  if [ ! -z "$VALIDATOR_AUTO_CONFIG" ] && [ "$VALIDATOR_AUTO_CONFIG" = "1" ]; then
    if [ -z "$VALIDATOR_KEYNAME" ] || [ -z "$VALIDATOR_MNENOMIC" ] || [ -z "$VALIDATOR_PASSPHRASE" ] ; then
      echo "VALIDATOR_AUTO_CONFIG=1 requires VALIDATOR_KEYNAME, VALIDATOR_MNENOMIC, and VALIDATOR_PASSPHRASE" >&2
      exit 1
    fi

    if [ ! -z "$VALIDATOR_KEYNAME" ] && [ ! -z "$VALIDATOR_MNENOMIC" ] && [ ! -z "$VALIDATOR_PASSPHRASE" ] ; then
      dochaind keys add $VALIDATOR_KEYNAME --recover > ~/.do/keys.log 2>&1 << EOF
$VALIDATOR_MNENOMIC
$VALIDATOR_PASSPHRASE
$VALIDATOR_PASSPHRASE
EOF
    fi

    if [ ! -z "$VALIDATOR_AMOUNT" ] && [ ! -z "$MONIKER" ] && [ ! -z "$VALIDATOR_PASSPHRASE" ] && [ ! -z "$VALIDATOR_KEYNAME" ] && [ ! -z "$VALIDATOR_KEYNAME" ] && [ ! -z "$VALIDATOR_COMMISSION_RATE" ] && [ ! -z "$VALIDATOR_COMMISSION_RATE_MAX" ]  && [ ! -z "$VALIDATOR_COMMISSION_RATE_MAX_CHANGE" ]  && [ ! -z "$VALIDATOR_MIN_SELF_DELEGATION" ] ; then
      dochaind tx staking create-validator --amount=$VALIDATOR_AMOUNT --pubkey=$(dochaind tendermint show-validator) --moniker="$MONIKER" --chain-id=$CHAINID --from=$VALIDATOR_KEYNAME --commission-rate="$VALIDATOR_COMMISSION_RATE" --commission-max-rate="$VALIDATOR_COMMISSION_RATE_MAX" --commission-max-change-rate="$VALIDATOR_COMMISSION_RATE_MAX_CHANGE" --min-self-delegation="$VALIDATOR_MIN_SELF_DELEGATION" --gas=$VALIDATOR_GAS --gas-adjustment=$VALIDATOR_GAS_ADJUSTMENT --fees=$VALIDATOR_FEES > ~/.do/validator.log 2>&1 << EOF
$VALIDATOR_PASSPHRASE
y
EOF
    fi
  fi
fi

wait





