package httpmock

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tj/go-spin"
)

const (
	DEFAULT_PORT      = "6379"
	SERVER_ADDR       = "localhost:" + DEFAULT_PORT
	NETWORK           = "tcp"
	SEPARATOR         = ","
	CONNECT_THRESHOLD = 1000
)

var lastEof int64 = 0
var isWaitingForConnection bool = true

func Connect(mockString []string) {
	// listenSignal()
	spin := spin.New()

	go detectConnected()

	for {
		fmt.Printf("\r  \033[36mConnecting\033[m %s ", spin.Next())

		tcpAddr, err := net.ResolveTCPAddr(NETWORK, SERVER_ADDR)
		CheckErr(err)

		conn, err := net.DialTCP(NETWORK, nil, tcpAddr)
		CheckErr(err)
		defer conn.Close()

		write(conn, createPayload(mockString))

		reader := bufio.NewReader(conn)
		handleResponseWithReader(*reader)

		// Make things slower
		time.Sleep(200 * time.Millisecond)
	}
}

func createPayload(mockString []string) string {
	return strings.Join(mockString, SEPARATOR)
}

func handleResponseWithReader(reader bufio.Reader) {
	lastEof = makeTimestamp()
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				lastEof = makeTimestamp()
				break
			}
		}
		println("=>", string(line))
	}
}

func write(conn net.Conn, content string) (int, error) {
	writer := bufio.NewWriter(conn)
	number, err := writer.WriteString(content + "\n")
	if err == nil {
		err = writer.Flush()
	}
	return number, err
}

func clearMock() {
	println("Clear mocks!")
}

func detectConnected() {
	for {
		if lastEof == 0 {
			continue
		}

		current := makeTimestamp()
		diff := current - lastEof

		time.Sleep(500 * time.Millisecond)

		if diff > CONNECT_THRESHOLD {
			fmt.Printf("\r\n  \033[36mConnected. Waiting for requests…\033[m\n\n")
			lastEof = 0
		}
	}
}

func listenSignal() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println("Signal:")
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
