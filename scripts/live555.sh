#!/usr/bin/env bash

HERE=$(readlink -f "$(dirname "$0")")
# shellcheck source=common.bash
source "$HERE/common.bash"
check_help "$@"

# kill old instance, clean gcda and fuzzing folder
killall -9 testOnDemandRTSPServer > /dev/null 2>&1
gcovr -r /work/live555 -d > /dev/null 2>&1

set -e
timeout -s SIGUSR1 3s /work/live555/testProgs/testOnDemandRTSPServer 8554 2> /dev/null &
serverpid=$!

/work/tester -dir=msgs -host=localhost:8554 "$@" &
clientpid=$!

set +e
wait "$serverpid"
kill "$clientpid" > /dev/null 2>&1

dump_coverage /work/live555
check_line "DESCRIBE" "liveMedia/RTSPServer.cpp" 321
check_line "SETUP" "liveMedia/RTSPServer.cpp" 1249
check_line "PLAY" "liveMedia/RTSPServer.cpp" 1604
check_line "TEARDOWN" "liveMedia/RTSPServer.cpp" 1574
