#!/bin/bash

#set -e -o pipefail

P="\033["
BLUE=34
YELLOW=33
RED=31
END="0m"



function clear_line() {
  local num=$1
  local i=0
  while true
  do
     if [ $i -eq $num ]; then
         return
     fi
     printf "\033[A\33[2KT\r"
     let i=$(expr "$i+1")
  done
}

function show_statistics() {
  local json=$1
  server=$(echo $json |jq -r '.[0].server')
  mac=$(echo $json |jq -r '.[0].mac')
  ifindex=$(echo $json |jq '.[0].ifindex')
  forward_bytes=$(echo $json |jq '.[0].forward_bytes')
  forward_packages=$(echo $json |jq '.[0].forward_packages')
  date=$(date "+%Y-%d-%m_%H:%M:%S")

  clear_line 3

  printf "%30s: %b%s%b\n" "Timestamp" ${P}${RED}"m" ${date} ${P}${END}
  printf "%-25s%-25s%10s%20s%20s\n" "Server" "MAC" "ifindex" "Forward Bytes" "Forward Packages"
  printf "%-25s%-25s%10s%b%20s%b%b%20s%b\n" ${server} ${mac} ${ifindex} ${P}${BLUE}"m" ${forward_bytes} ${P}${END} ${P}${YELLOW}"m" ${forward_packages} ${P}${END}

}

function fetch_statistics() {
  local host=$1
  local port=$2

  local output=$(curl http://${host}:${port}/rules 2>/dev/null)
  if [ ${output:-x} == "x" ];then
    echo "Accessing the webserver failed"
    return 2
  fi
  echo $output
}



function usage() {
   local prog=$(basename ${BASH_SOURCE[0]})
   echo "${prog} show the statistics of packets redirecting "
   echo "usage:" 
   echo "${prog} [-h HOST][-p PORT]"
   echo "-h HOST"
   echo "     - address on which the webserver listened, default is 127.0.0.1"
   echo "-h PORT"
   echo "     - port on which the webserver listened, default is 9091"
}

PORT=9091
HOST=127.0.0.1
while getopts "h:p:" opt; do
  case ${opt} in
    h) 
      HOST=${OPTARG}
      ;;
    p)
      PORT=${OPTARG}
      ;;
    \?) 
     usage
     exit 2;
      ;;
  esac
done

clear
while true
do
  OUTPUT=$(fetch_statistics "$HOST" "${PORT}")
  if [ $? -ne 0 ]; then
     echo ${OUTPUT}
     exit 2
  fi
  show_statistics "${OUTPUT}"
  sleep 1
done
