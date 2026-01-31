#!/bin/bash

# Easily send a data file to the remote host

prefix=$(grep 'SubsurfaceDataFilePrefix' "$(dirname -- "${BASH_SOURCE[0]}")/../server/constants.go" | cut -d'"' -f2)
selected="$(find "${DIVELOG_LOCAL_BACKUP_DIR}" -maxdepth 1 -type f -name "${prefix}*.xml" | fzf)"
selected_path="$(realpath "${selected}")"
selected_file="$(basename "${selected}")"

echo "Sending ${selected_path} to the remote target ${DIVELOG_SSH_LOGIN_TARGET} ..."
rsync -vc "${selected_path}" "${DIVELOG_SSH_LOGIN_TARGET}:${DIVELOG_HOST_STORE_DIR}/${selected_file}"
