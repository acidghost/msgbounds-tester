#!/usr/bin/env bash

# kill old instance, clean gcda and fuzzing folder
killall -9 pure-ftpd > /dev/null 2>&1
gcovr -r /work/pure-ftpd -s > /dev/null 2>&1
rm -rf /home/fuzzing/*

# start server
GCOV_PREFIX=/home/fuzzing /work/pure-ftpd/src/pure-ftpd -A &
serverpid=$!

# start client
/work/tester -host=localhost:21 \
    -init-sleep=0 \
    -init-read=false \
    -sleep=0 \
    -read=false \
    -fin-sleep=5s \
    "$@" &
clientpid=$!

# wait for the server to exit and kill the client
wait "$serverpid"
kill -9 "$clientpid"

# copy gcda from chrooted folder to source folder
set -e
cp /home/fuzzing/work/pure-ftpd/src/*.gcda /work/pure-ftpd/src
gcovr -r /work/pure-ftpd -s | grep branch
gcovr -r /work/pure-ftpd --json > /work/cov.json
set +e

function check_line {
    local file=$1 ln=$2
    local p=".files[] | select(.file == \"$file\") \
        | .lines[] | select(.line_number == $ln) | .count"
    local res
    if ! res=$(jq "$p" /work/cov.json); then
        echo "!!! Error running jq"
        return 1
    fi
    if [ "$res" = "0" ]; then
        echo "Line $ln NOT covered"
    else
        echo "Line $ln covered"
    fi
}

# received "list" command
check_line "src/ftp_parser.c" 619
# received "quit" command
check_line "src/ftp_parser.c" 344
