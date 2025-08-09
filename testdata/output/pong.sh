#!/bin/bash
count=0
trap sigusr1_handler SIGUSR1

function sigusr1_handler() {
    stdbuf -o0 echo "PING $((++count))"
    stdbuf -o0 kill -SIGUSR1 $pids 2>/dev/null
}

while [[ -z $pids ]]
do   
    pids=$(ps -ef | grep ping.sh | grep -v grep | awk '{print $2}')
    sleep 0.001
done
 
while [[ $count -lt 10 ]]; do
    sleep 0.001
done
