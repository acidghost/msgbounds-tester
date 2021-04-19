# msgbounds-tester
Test how servers respond to different ways of interacting with it such as:
- reading replies or not;
- reading replies thoroughly or not;
- different read timeout;
- amount of delay between sent messages (if any).

## Usage
Build container image for desired target with `make $target`.
Start container (e.g. `docker run --rm msgbounds:$target -h` for help).

Start the container with environment variable `DO_STRACE=1` to run the tester (and target) under
`strace` to trace some useful syscalls. The output is stored in `/work/strace.log`.
