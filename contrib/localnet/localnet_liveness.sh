#!/usr/bin/env bash

set -u

CNT=0
ITER=${1:-}
SLEEP=${2:-}
NUMBLOCKS=${3:-}
NODEADDR=${4:-}

if [ -z "${ITER}" ]; then
  echo "Need to input number of iterations to run..."
  exit 1
fi

if [ -z "${SLEEP}" ]; then
  echo "Need to input number of seconds to sleep between iterations"
  exit 1
fi

if [ -z "${NUMBLOCKS}" ]; then
  echo "Need to input block height to declare completion..."
  exit 1
fi

if [ -z "${NODEADDR}" ]; then
  echo "Need to input node address to poll..."
  exit 1
fi

mapfile -t docker_containers < <(docker compose ps -q)
if [ "${#docker_containers[@]}" -eq 0 ]; then
  echo "No localnet containers found."
  exit 1
fi

while [ "${CNT}" -lt "${ITER}" ]; do
  curr_block=$(curl --fail --silent --max-time 2 "http://${NODEADDR}:26657/status" 2>/dev/null | jq -er '.result.sync_info.latest_block_height' 2>/dev/null || true)

  if [[ "${curr_block}" =~ ^[0-9]+$ ]]; then
    echo "Number of Blocks: ${curr_block}"
  fi

  if [[ "${curr_block}" =~ ^[0-9]+$ ]] && [ "${curr_block}" -gt "${NUMBLOCKS}" ]; then
    echo "Number of blocks reached. Success!"
    exit 0
  fi

  # Emulate network chaos:
  #
  # Every 10th iteration, pick a random container and restart it.
  if ((CNT > 0 && CNT % 10 == 0)); then
    rand_container=${docker_containers[$((RANDOM % ${#docker_containers[@]}))]}
    echo "Restarting random docker container ${rand_container}"
    docker restart "${rand_container}" &>/dev/null &
  fi
  CNT=$((CNT + 1))
  sleep "${SLEEP}"
done
echo "Timeout reached. Failure!"
exit 1






