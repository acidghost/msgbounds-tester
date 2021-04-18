// Test how a server responds to different ways of interacting with it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	flagHost      = flag.String("host", "172.17.0.2:21", "host to connect to")
	flagDir       = flag.String("dir", "messages/ftp", "folder with messages to send")
	flagRead      = flag.Bool("read", false, "read server replies")
	flagSimpRead  = flag.Bool("simp", false, "read with a single call")
	flagTimeout   = flag.Duration("time", 30*time.Millisecond, "read deadline")
	flagInitRead  = flag.Bool("init-read", false, "do initial read (e.g. banner message)")
	flagInitSleep = flag.Duration("init-sleep", 10*time.Millisecond, "sleep after connection")
	flagSleep     = flag.Duration("sleep", 1*time.Millisecond, "sleep after each send")
	flagFinSleep  = flag.Duration("fin-sleep", 3*time.Second, "sleep before closing")
	flagSignal    = flag.Int("signal", int(unix.SIGTERM), "signal to stop the server")
)

func main() {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s [flags] [target [args...]]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	log.SetFlags(log.Lmicroseconds)

	stopSignal := syscall.Signal(*flagSignal)

	msgs := loadMessages(*flagDir)

	var serv *server
	if flag.NArg() > 0 {
		args := flag.Args()
		cmd := exec.Command(args[0], args[1:]...)
		stdout, err := cmd.StdoutPipe()
		log.Printf("Starting server %q\n", cmd)
		if err != nil {
			log.Fatalf("Failed to get server's stdout: %v\n", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Failed to get server's stderr: %v\n", err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
		serv = &server{cmd, stdout, stderr}
	}

	signChan := make(chan os.Signal)
	go func() {
		<-signChan
		log.Printf("Signaled to quit\n")
		os.Exit(0)
	}()
	signal.Notify(signChan, unix.SIGTERM)

	log.Printf("Connecting to %s...\n", *flagHost)
	conn, err := connect(*flagHost)
	if err != nil {
		log.Fatalf("Could not connect: %v\n", err)
	}
	defer conn.Close()

	time.Sleep(*flagInitSleep)

	recvMsg := func() {
		msg, err := recv(conn, *flagSimpRead)
		if err != nil {
			log.Printf("Error receiving: %v\n", err)
		} else {
			log.Printf("Read: %s\n", msg)
		}
	}

	if *flagInitRead {
		log.Printf("Doing initial read\n")
		recvMsg()
	}

	for i, msg := range msgs {
		log.Printf("Sending %d: %s\n", i, ppMsg(msg))
		n, err := conn.Write(msg)
		if err != nil {
			log.Fatalf("Failed to send message %d: %v\n", i, err)
		}
		if n < len(msg) {
			log.Printf("Sent less bytes than expected: %d instead of %d\n", n, len(msg))
		}

		time.Sleep(*flagSleep)

		if *flagRead {
			recvMsg()
		}
	}

	if serv == nil {
		time.Sleep(*flagFinSleep)
	} else {
		exitCode, needWait := 0, true
		select {
		case <-time.After(*flagFinSleep):
			log.Printf("Reached timeout before server exited\n")
			serv.stop(stopSignal)
		case ws := <-serv.wait():
			exitCode = ws.ExitStatus()
			needWait = false
		}
		serv.Stdout()
		serv.Stderr()
		if needWait {
			var exitErr *exec.ExitError
			if err := serv.cmd.Wait(); err != nil && !errors.As(err, &exitErr) {
				log.Fatalf("Failed to wait for server's termination: %v\n", err)
			}
			exitCode = serv.cmd.ProcessState.ExitCode()
		}
		log.Printf("Server's exit code: %d\n", exitCode)
	}
}

func loadMessages(dir string) [][]byte {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read dir: %v\n", err)
	}
	msgs := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		name := filepath.Join(dir, entry.Name())
		msg, err := os.ReadFile(name)
		if err != nil {
			log.Fatalf("Failed to read message from file: %v\n", err)
		}
		log.Printf("Loaded message %s (%d bytes)\n", name, len(msg))
		msgs = append(msgs, msg)
	}
	return msgs
}

type server struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (s *server) wait() <-chan unix.WaitStatus {
	c := make(chan unix.WaitStatus)
	go func() {
		var ws unix.WaitStatus
		for {
			pid, err := unix.Wait4(s.cmd.Process.Pid, &ws, unix.WNOHANG, nil)
			if err != nil {
				log.Fatalf("Failed to wait for server PID %d: %v\n", s.cmd.Process.Pid, err)
			} else if pid == s.cmd.Process.Pid && (ws.Exited() || ws.Signaled()) {
				c <- ws
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return c
}

func (s *server) stop(signal os.Signal) {
	log.Printf("Stopping server...\n")
	if err := s.cmd.Process.Signal(signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Printf("Could not kill server (PID %d): %v\n", s.cmd.Process.Pid, err)
	}
}

func (s *server) Stdout() {
	s.output(false)
}

func (s *server) Stderr() {
	s.output(true)
}

func (s *server) output(stderr bool) {
	var r io.ReadCloser
	var str string
	if stderr {
		r, str = s.stderr, "stderr"
	} else {
		r, str = s.stdout, "stdout"
	}
	out, err := io.ReadAll(r)
	if err == nil {
		if len(out) > 0 {
			log.Printf("Server's %s:\n%s\n", str, out)
		} else {
			log.Printf("Server's %s: n/a\n", str)
		}
	} else {
		log.Printf("Could not read server's %s: %v\n", str, err)
	}
}

func connect(host string) (c net.Conn, err error) {
	const T time.Duration = 6 * time.Second
	timeout := time.After(T)
	for {
		c, err = net.Dial("tcp", host)
		if err == nil {
			return
		}
		select {
		case <-timeout:
			log.Fatalf("Timeout trying to connect to %q (%s)\n", host, T)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func ppMsg(msg []byte) string {
	const MAX int = 50
	trunc := false
	l := len(msg)
	if l > MAX {
		trunc = true
		l = MAX
	}
	replacer := strings.NewReplacer("\r", "\\r", "\n", "\\n", "\t", "\\t")
	s := replacer.Replace(string(msg[:l]))
	if trunc {
		s += "..."
	}
	return s
}

func recv(c net.Conn, simple bool) (buf []byte, err error) {
	buf = make([]byte, 4096)
	start := 0
	for {
		var n int
		resetDeadline(c)
		n, err = c.Read(buf[start:])
		log.Printf("Read %d bytes\n", n)
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				log.Printf("Timed out\n")
				err = nil
			} else if errors.Is(err, io.EOF) {
				log.Printf("EOF\n")
				err = nil
			}
			break
		}
		if n == 0 {
			buf = buf[:start]
			break
		}
		start += n
		if simple {
			buf = buf[:start]
			break
		}
		if start >= 4096 {
			break
		}
	}
	return
}

func resetDeadline(c net.Conn) {
	if err := c.SetReadDeadline(time.Now().Add(*flagTimeout)); err != nil {
		log.Printf("Failed to set timeout on socket: %v\n", err)
	}
}
