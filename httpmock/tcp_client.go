package httpmock

import (
	"bufio"
	"io"
	"net"
	"strings"
)

const (
	DEFAULT_PORT = "6379"
	SERVER_ADDR  = "localhost:" + DEFAULT_PORT
	NETWORK      = "tcp"
	SEPARATOR    = ","
)

func Connect(mockString []string) {
	tcpAddr, err := net.ResolveTCPAddr(NETWORK, SERVER_ADDR)
	CheckErr(err)

	conn, err := net.DialTCP(NETWORK, nil, tcpAddr)
	CheckErr(err)
	defer conn.Close()

	write(conn, createPayload(mockString))
	conn.CloseWrite()

	reader := bufio.NewReader(conn)
	handleResponseWithReader(*reader)
}

func createPayload(mockString []string) string {
	return strings.Join(mockString, SEPARATOR)
}

func handleResponseWithReader(reader bufio.Reader) {
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		println("=>", string(line))
	}
}

func write(conn net.Conn, content string) (int, error) {
	writer := bufio.NewWriter(conn)
	number, err := writer.WriteString(content)
	if err == nil {
		err = writer.Flush()
	}
	return number, err
}
