#!/bin/bash

P="\033["
BLUE=34
YELLOW=33
END="0m"

PORT=9091

function output() {
  local json=$1
  server=$(echo $json |jq -r '.[0].server')
  mac=$(echo $json |jq -r '.[0].mac')
  ifindex=$(echo $json |jq '.[0].ifindex')
  forward_bytes=$(echo $json |jq '.[0].forward_bytes')
  forward_packages=$(echo $json |jq '.[0].forward_packages')
  date=$(date "+%Y-%d-%m_%H:%M:%S")

  printf "\033[A\33[2KT\r"
  printf "\033[A\33[2KT\r"
  printf "\033[A\33[2KT\r"
  printf "%30s: %s\n" "Timestamp" ${date}
  printf "%-25s%-25s%10s%20s%20s\n" "Server" "MAC" "ifindex" "Forward Bytes" "Forward Packages"
  printf "%-25s%-25s%10s%b%20s%b%b%20s%b\n" ${server} ${mac} ${ifindex} ${P}${BLUE}"m" ${forward_bytes} ${P}${END} ${P}${YELLOW}"m" ${forward_packages} ${P}${END}
}

clear

while true
do
sleep 1
OUTPUT=$(curl http://localhost:${PORT}/rules 2>/dev/null)
output $OUTPUT
done
