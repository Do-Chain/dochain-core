# do Core Dockerized

These are legacy operator examples. Do not use them as production Do-Chain
node configuration without reviewing the chain ID, image tag, genesis source,
and exposed ports.

## Common usage examples 

### Mainnet 

Standard options with LCD enabled: 

```
docker run -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle
```

LCD disabled: 

```
docker run -e ENABLE_LCD=false -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle
```

Custom gas fees: 

```
docker run -e MINIMUM_GAS_PRICES="0.01133udo,0.15uusd,0.104938usdr,169.77ukrw,428.571umnt,0.125ueur,0.98ucny,16.37ujpy,0.11ugbp,10.88uinr,0.19ucad,0.14uchf,0.19uaud,0.2usgd,4.62uthb,1.25usek,1.25unok,0.9udkk,2180.0uidr,7.6uphp,1.17uhkd" -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle
```

Starting the sync from a snapshot:

```
docker run -e SNAPSHOT_NAME="cookie-1-pruned.20211022.0410.tar.lz4" -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle
```

You can find the latest snapshots [here](https://quicksync.io/networks/do.html).

Custom snapshot URL:

```
docker run -e SNAPSHOT_BASE_URL="https://get.quicksync.io" -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle
```

**Note:** We recommend copying a snapshot to S3 or another file store and using the above options to point the container to your snapshot. The default snapshot name included will be obsolete and removed in a matter of days.

Starting a bombay node: 

```
docker run -it -p 127.0.0.1:1317:1317 -p 127.0.0.1:26657:26657 -p 26656:26656 dochain/core-node:v0.5.11-oracle-testnet
```

Bind LCD and RPC to `127.0.0.1` unless the node is deliberately fronted by a
firewall or rate-limited reverse proxy. Leave only the P2P port public by
default.

Legacy remote genesis and addrbook downloads are disabled by default. Set
`ALLOW_REMOTE_NETWORK_CONFIG=true` only when you trust the referenced source and
have verified the expected genesis.

## Building the Docker images

```
./build_all.sh v0.5.11-oracle
```





