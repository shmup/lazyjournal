#!/bin/bash

debug=$1
exitCode=0

sudo systemctl restart cron
echo -e "First line\nSecond line" > input.log

tmux new-session -d -s test-session "go run main.go"

# Finally
trap "
    tmux kill-session -t test-session
    rm *.log
    exit \$exitCode
" EXIT

# Test 1
sleep 2
tmux send-keys -t test-session "cron"
tmux send-keys -t test-session "$(echo -e '\t')"
tmux send-keys -t test-session "$(echo -e '\x1b[C')" 
sleep 2
tmux send-keys -t test-session "$(echo -e '\r')"
sleep 2
tmux send-keys -t test-session "$(echo -e '\t\t\t')"
tmux send-keys -t test-session "$(echo -e '\x1b[A')"
tmux send-keys -t test-session "started"
sleep 2
if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
output=$(tmux capture-pane -p)

if echo "$output" | grep -q "Started cron"; then
    echo "✔  Test read journal from systemd-journald: Passed"
else
    echo "❌ Test read journal from systemd-journald: Failed"
    exitCode=1
fi

for i in {1..7}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done
tmux send-keys -t test-session "$(echo -e '\t\t')"
for i in {1..4}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done

# Test 2
tmux send-keys -t test-session "input"
tmux send-keys -t test-session "$(echo -e '\t\t')"
tmux send-keys -t test-session "$(echo -e '\x1b[C')"
sleep 3
tmux send-keys -t test-session "$(echo -e '\r')"
sleep 3
if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
output=$(tmux capture-pane -p)

if echo "$output" | grep -q "First line" && echo "$output" | grep -q "Second line"; then
    echo "✔  Test read file: Passed"
else
    echo "❌ Test read file: Failed"
    exitCode=2
fi

tmux send-keys -t test-session "$(echo -e '\t\t\t\t')"
for i in {1..5}; do tmux send-keys -t test-session "$(echo -e '\x7f')"; done

# Test 3
tmux send-keys -t test-session "ping"
tmux send-keys -t test-session "$(echo -e '\t\t\t')"
tmux send-keys -t test-session "$(echo -e '\r')"
sleep 2
tmux send-keys -t test-session "$(echo -e '\t')"
tmux send-keys -t test-session "http"
sleep 2
if [ "$debug" == "true" ]; then tmux capture-pane -p; fi
output=$(tmux capture-pane -p)

if echo "$output" | grep -q "The server is running on http://localhost:3005" && echo "$output" | grep -q "Local: http://localhost:8085"; then
    echo "✔  Test read log from docker container: Passed"
else
    echo "❌ Test read log from docker container: Failed"
    exitCode=3
fi
