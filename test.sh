#!/bin/bash

# bash test.sh <timeout> <log>
# bash test.sh 3 true

timeout=${1:-5}
debug=$2

exitCode=0

sudo systemctl restart cron
echo -e "First line\nSecond line" > input.log

filterContainer="pinguem"
filterLog="running"
searchText="The server is running"

journalctlVersion=$(journalctl --version journalctl --version 2> /dev/null || echo false)
dockerVersion=$(docker --version 2> /dev/null || echo false)
dockerContainer=$(docker ps | grep $filterContainer 2> /dev/null || echo false)

tmux new-session -d -s test-session "go run main.go"
sleep 1

# Finally
trap "
    tmux kill-session -t test-session
    rm *.log
    exit \$exitCode
" EXIT

# Test 1
if [ "$journalctlVersion" == "false" ]; then
    echo "Test journald: 🚫 Skip (journalctl not found)"
else
    tmux send-keys -t test-session "cron"
    tmux send-keys -t test-session "$(echo -e '\t')"
    tmux send-keys -t test-session "$(echo -e '\x1b[C')" 

    start_time=$(date +%s)
    while true; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if tmux capture-pane -p | grep -q "< System j" || [ "$elapsed" -ge "$timeout" ]; then
            break
        fi
    done

    tmux send-keys -t test-session "$(echo -e '\r')"
    tmux send-keys -t test-session "$(echo -e '\t\t\t')"
    tmux send-keys -t test-session "$(echo -e '\x1b[A')"
    tmux send-keys -t test-session "started"

    start_time=$(date +%s)
    while true; do
        if tmux capture-pane -p | grep -q "systemd\[1\]: Started"; then
            echo "Test journald: ✔  Passed"
            break
        fi
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if [ "$elapsed" -ge "$timeout" ]; then
            echo "Test journald: ❌ Failed"
            exitCode=1
            break
        fi
    done

    if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
    for i in {1..7}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done
    tmux send-keys -t test-session "$(echo -e '\t\t')"
    for i in {1..4}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done
fi

# Test 2
tmux send-keys -t test-session "input"
tmux send-keys -t test-session "$(echo -e '\t\t')"
tmux send-keys -t test-session "$(echo -e '\x1b[C')"

start_time=$(date +%s)
while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    if [[ $(tmux capture-pane -p | grep -q "< Users h") && $(tmux capture-pane -p | grep -q -v "Searching") ]] || [ "$elapsed" -ge "$timeout" ]; then
        break
    fi
done

tmux send-keys -t test-session "$(echo -e '\r')"

start_time=$(date +%s)
while true; do
    if tmux capture-pane -p | grep -q "First line" && tmux capture-pane -p | grep -q "Second line"; then
        echo "Test file: ✔  Passed"
        break
    fi
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    if [ "$elapsed" -ge "$timeout" ]; then
        echo "Test file: ❌ Failed"
        exitCode=2
        break
    fi
done

if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
tmux send-keys -t test-session "$(echo -e '\t\t\t\t')"
for i in {1..5}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done

# Test 3
if [ "$dockerVersion" == "false" ]; then
    echo "Test docker: 🚫 Skip (not installed)"
elif [ "$dockerContainer" == "false" ]; then
    echo "Test docker: 🚫 Skip (container not found)"
else
    tmux send-keys -t test-session "$(echo -e '\t\t\t\t')"
    tmux send-keys -t test-session "$filterLog"
    tmux send-keys -t test-session "$(echo -e '\t\t')"
    tmux send-keys -t test-session "$filterContainer"
    tmux send-keys -t test-session "$(echo -e '\t\t\t')"
    tmux send-keys -t test-session "$(echo -e '\r')"

    start_time=$(date +%s)
    while true; do
        if tmux capture-pane -p | grep -q "$searchText"; then
            echo "Test docker: ✔  Passed"
            break
        fi
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if [ "$elapsed" -ge "$timeout" ]; then
            echo "Test docker: ❌ Failed"
            exitCode=3
            break
        fi
    done

    if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
fi
