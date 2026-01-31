#!/bin/bash

# Easily setup and run the server locally.

msg() {
  echo "------------------------------------------------------"
  echo " $1"
  echo "------------------------------------------------------"
}

export DIVELOG_DBFILE_PATH="${DIVELOG_LOCAL_BACKUP_DIR}/subsurface-2025-12.xml"
export DIVELOG_MODE=dev

msg "Dumping local env state ..."
$(dirname -- "${BASH_SOURCE[0]}")/env.sh

msg "Starting the server ..."
set -x
LOGFILE="bluefin.log"
go run ./main.go > "${LOGFILE}" 2>&1 &
SERVER_PID=$!

{ set +x; } 2>/dev/null
# Inside a tmux session, tail a log in a new/existing pane to the right.
if [ -n "${TMUX:-}" ]; then
    if ! tmux list-panes -F '#{pane_title}' | grep -qx "tail-log-pane"; then
        tmux split-window -d -h -c "#{pane_current_path}" "tail -n 100 -F '${LOGFILE}'"
        tmux select-pane -T "tail-log-pane" -t "{right-of}"
    fi
else
    msg "Logs have been redirected to ${LOGFILE}"
fi
set -x

# Forward CTRL-C to the server process
trap "kill -INT ${SERVER_PID}" INT

sleep 1

{ set +x; } 2>/dev/null
msg "Waiting for pid ${SERVER_PID} ..."
set -x
wait $SERVER_PID
