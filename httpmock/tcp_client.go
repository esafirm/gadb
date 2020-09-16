package httpmock

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
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
var connection net.Conn

func Connect(mockString []string) {
	spin := spin.New()

	go listenSignal()
	go detectConnected()

	for {
		fmt.Printf("\r  \033[36mConnecting\033[m %s ", spin.Next())

		tcpAddr, err := net.ResolveTCPAddr(NETWORK, SERVER_ADDR)
		CheckErr(err)

		connection, err = net.DialTCP(NETWORK, nil, tcpAddr)
		CheckErr(err)
		defer connection.Close()

		write(connection, createMockPayload(mockString))

		reader := bufio.NewReader(connection)
		handleResponseWithReader(*reader)

		// Make things slower
		time.Sleep(200 * time.Millisecond)
	}
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
	println("Clearing mock…")
	write(connection, createClearPayload())
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

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println(sig)
		done <- true
	}()

	<-done
	clearMock()
	os.Exit(0)
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
