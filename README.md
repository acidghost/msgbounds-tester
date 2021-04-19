# msgbounds-tester
Test how servers respond to different ways of interacting with it such as:
- sending multiple messages at once (i.e. as a single syscall);
- reading replies or not;
- reading replies thoroughly or not;
- different read timeout;
- amount of delay between sent messages (if any);
- closing the socket right after sending the last message or after the server exits.

## Usage
Build container image for desired target with `make $target`.
Start container (e.g. `docker run --rm msgbounds:$target -h` for help).

Start the container with environment variable `DO_STRACE=1` to run the tester (and target) under
`strace` to trace some useful syscalls. The output is stored in `/work/strace.log`.

The scripts should also produce the coverage of the target into `/work/cov.json` (from gcovr).
