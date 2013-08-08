package main

import (
	"bytes"
	"strconv"
	"net"
	"time"
	"log"
	"strings"
)

const (
	PORT = 4573
	RECV_BUF_LEN = 4096
)

func main() {
	log.Println("Starting FastAGI server...")

	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(PORT))
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Connected: %v <-> %v\n", conn.LocalAddr(), conn.RemoteAddr())
		msgchan := make(chan string)
		go agi_conn_handle(conn, msgchan)
		go agi_parse(msgchan)
	}
}

func agi_conn_handle(client net.Conn, msgchan chan<- string) {
	for {
		buf := make([]byte, RECV_BUF_LEN)
		client.SetReadDeadline(time.Now().Add(20 * time.Second))

		n, err := client.Read(buf)
		if err != nil || n == 0 {
			log.Println(err)
			break
		}
		//log.Println(string(buf))

		if bytes.Contains(buf, []byte("HANGUP")) {
			log.Println("Hanging up client.")
			break
		}
		msgchan <- string(buf[0:n])
	}
	log.Printf("Connection from %v closed.", client.RemoteAddr())
	client.Close()
	return
}

func agi_parse(msgchan <-chan string) {
	for msg := range msgchan {
		input_str := strings.SplitN(msg, ": ", 2)
		if len(input_str) == 2 {
			input_str[0] = strings.TrimPrefix(input_str[0], "agi_")
			input_str[1] = strings.TrimRight(input_str[1], "\n")
			log.Printf("AGI input: %s -> %s", input_str[0], input_str[1])
		}
	}
}
