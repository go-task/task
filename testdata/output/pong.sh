#!/bin/bash
ready=0
count=0
trap sigusr1_handler SIGUSR1
trap sigusr2_handler SIGUSR2

function sigusr1_handler() {
    ((ready = 1))
}

function sigusr2_handler() {
    ((count++))
    stdbuf -o0 echo "PING $count"
    if [ $count -lt 11 ]; then
        stdbuf -o0 kill -SIGUSR2 $pids 2>/dev/null
    fi
}

while [[ -z $pids ]]
do   
    pids=$(ps -ef | grep ping.sh | grep -v grep | awk '{print $2}')
    sleep 0.001
done

while [[ $ready == 0 ]]
do
    stdbuf -o0 kill -SIGUSR1 $pids 2>/dev/null
    sleep 0.001
done
 
while [[ $count -lt 10 ]]; do
    sleep 0.001
done
