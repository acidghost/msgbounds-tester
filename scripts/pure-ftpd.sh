#!/usr/bin/env bash

HERE=$(readlink -f "$(dirname "$0")")
# shellcheck source=common.bash
source "$HERE/common.bash"
check_help "$@"

# kill old instance, clean gcda and fuzzing folder
killall -9 pure-ftpd > /dev/null 2>&1
gcovr -r /work/pure-ftpd -d > /dev/null 2>&1
rm -rf /home/fuzzing/*

set -e
GCOV_PREFIX=/home/fuzzing timeout 5s /work/pure-ftpd/src/pure-ftpd -A &
serverpid=$!

/work/tester -dir=msgs -host=localhost:21 "$@" &
clientpid=$!

set +e
wait "$serverpid"
kill "$clientpid" > /dev/null 2>&1

# copy gcda from chrooted folder to source folder
cp /home/fuzzing/work/pure-ftpd/src/*.gcda /work/pure-ftpd/src
dump_coverage /work/pure-ftpd
check_line "LIST" "src/ftp_parser.c" 619
check_line "QUIT" "src/ftp_parser.c" 344
