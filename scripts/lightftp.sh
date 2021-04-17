#!/usr/bin/env bash

HERE=$(readlink -f "$(dirname "$0")")
# shellcheck source=common.bash
source "$HERE/common.bash"
check_help "$@"

# kill old instance, clean gcda and fuzzing folder
killall -9 fftp > /dev/null 2>&1
gcovr -r /work/LightFTP -d > /dev/null 2>&1
rm -rf /home/fuzzing/*
if ! (mkdir /home/fuzzing/ftpshare && chown fuzzing:fuzzing /home/fuzzing/ftpshare); then
    echo "Failed to reset ftpshare"
    exit 1
fi

set -e
timeout 5s /work/LightFTP/Source/Release/fftp /work/lightftp.conf 2200 &
serverpid=$!

# XXX: DialTimeout is not working in go tester
sleep 1s
/work/tester -host=localhost:2200 "$@" &
clientpid=$!

set +e
wait "$serverpid"
kill "$clientpid" > /dev/null 2>&1

dump_coverage /work/LightFTP
check_line "LIST" "Source/Release/ftpserv.c" 714
check_line "QUIT" "Source/Release/ftpserv.c" 435
