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

run_tester -dir=msgs -host=localhost:2200 "$@" -- \
    /work/LightFTP/Source/Release/fftp /work/lightftp.conf 2200

dump_coverage /work/LightFTP
check_line "LIST" "Source/Release/ftpserv.c" 714
check_line "QUIT" "Source/Release/ftpserv.c" 435
