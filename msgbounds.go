// Test how a server responds to different ways of interacting with it.
//
// Some sample parameters with the objective of making the interaction as fast as possible:
// - pure-ftpd: -init-sleep=0 -init-read=false -sleep=0 -read=false -fin-sleep=1s
//              (waiting for the server to finish its stuff is important, otherwise it'll fail
//              writing to the socket and exit before processing the other requests);
// - lightftp:  -init-sleep=0 -init-read=false -sleep=1us -read=false -fin-sleep=1s
//              (same as above, but also needs 1us of wait time to properly separate messages).
package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"time"
)

var (
	flagHost      = flag.String("host", "172.17.0.2:21", "host to connect to")
	flagRead      = flag.Bool("read", false, "read server replies")
	flagSimpRead  = flag.Bool("simp", false, "read with a single call")
	flagTimeout   = flag.Duration("time", 30*time.Millisecond, "read deadline")
	flagInitRead  = flag.Bool("init-read", false, "do initial read (e.g. banner message)")
	flagInitSleep = flag.Duration("init-sleep", 10*time.Millisecond, "sleep after connection")
	flagSleep     = flag.Duration("sleep", 1*time.Millisecond, "sleep after each send")
	flagFinSleep  = flag.Duration("fin-sleep", 3*time.Second, "sleep before closing")
)

var msgs = [...][]byte{
	[]byte("USER fuzzing\r\n"),
	[]byte("PASS fuzzing\r\n"),
	[]byte("LIST\r\n"),
	[]byte("QUIT\r\n"),
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lmicroseconds)

	log.Printf("Connecting to %s...\n", *flagHost)
	conn, err := net.Dial("tcp", *flagHost)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
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
		log.Printf("Sending %d: %s\n", i, msg[:len(msg)-2])
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

	// wait for server to exit before closing?
	time.Sleep(*flagFinSleep)
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
