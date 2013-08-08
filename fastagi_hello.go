package main

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	PORT         = 4573
	RECV_BUF_LEN = 4096
	TIMEOUT      = 30
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
		go agi_conn_handle(conn)
	}
}

func agi_conn_handle(client net.Conn) {
	msgchan := make(chan string)
	go agi_parse(msgchan)

	for {
		buf := make([]byte, RECV_BUF_LEN)
		client.SetReadDeadline(time.Now().Add(TIMEOUT * time.Second))

		n, err := client.Read(buf)
		if err != nil || n == 0 {
			log.Println(err)
			break
		}
		//log.Printf("Got %d bytes: %s", n, string(buf))
		if strings.Contains(string(buf[0:n]), "HANGUP") {
			log.Println("Client hung up.")
			break
		}
		msgchan <- string(buf[0:n])
	}
	log.Printf("Connection from %v closed.", client.RemoteAddr())
	close(msgchan)
	client.Close()
	return
}

func agi_parse(msgchan <-chan string) {
	agi_data := make(map[string]string)
P:
	for msg := range msgchan {
		for _, agi_str := range strings.SplitAfter(msg, "\n") {
			if len(agi_str) == 1 {
				break P
			}
			input_str := strings.SplitN(agi_str, ": ", 2)
			if len(input_str) == 2 {
				input_str[0] = strings.TrimPrefix(input_str[0], "agi_")
				input_str[1] = strings.TrimRight(input_str[1], "\n")
				agi_data[input_str[0]] = input_str[1]
			}
		}
	}
	log.Println("Finished reading AGI vars:")
	for key, value := range agi_data {
		log.Println(key + "\t\t" + value)
	}
}
