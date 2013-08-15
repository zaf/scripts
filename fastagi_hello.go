package main

import (
	"log"
	"net"
	"runtime"
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
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	rcv_chan := make(chan string)
	snd_chan := make(chan string)
	//go agi_parse(rcv_chan, snd_chan)
	go func(client net.Conn, rcv_chan chan<- string) {
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
			rcv_chan <- string(buf[0:n])
		}
		log.Printf("Connection from %v closed.", client.RemoteAddr())
		client.Close()
		close(rcv_chan)
		return
	}(client, rcv_chan)

	go func(snd_chan <-chan string) {
		for {
			select {
			case agi_msg, ok := <-snd_chan:
				if !ok {
					log.Printf("Channel closed.")
					return
				} else {
					_, err := client.Write([]byte(agi_msg))
					if err != nil {
						log.Println(err)
						return
					}
				}
			}
		}
	}(snd_chan)
	agi_parse(rcv_chan, snd_chan)
	return
}

func agi_parse(rcv_chan <-chan string, snd_chan chan<- string) {
	agi_data := make(map[string]string)
	reply := make([]string, 3)
LOOP:
	for msg := range rcv_chan {
		for _, agi_str := range strings.SplitAfter(msg, "\n") {
			if len(agi_str) == 1 {
				break LOOP
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

	if agi_data["arg_1"] == "" {
		log.Println("No arguments passed, exiting")
		goto HANGUP
	}

	snd_chan <- "VERBOSE \"Staring an echo test.\" 3\n"
	reply = agi_response(<-rcv_chan)

	//Check channel status and answer if not answered already
	snd_chan <- "CHANNEL STATUS\n"
	reply = agi_response(<-rcv_chan)
	if reply[1] == "4" {
		snd_chan <- "ANSWER\n"
		reply = agi_response(<-rcv_chan)
		if reply[1] == "-1" {
			log.Println("Failed to answer channel")
			goto HANGUP
		}
	}
	//Playback a file and run the echo() app
	snd_chan <- "STREAM FILE " + agi_data["arg_1"] + "  \"\"\n"
	reply = agi_response(<-rcv_chan)
	if reply[1] == "-1" {
		log.Println("Failed to playback file", agi_data["arg_1"])
	}
	snd_chan <- "EXEC echo\n"
	reply = agi_response(<-rcv_chan)
	if reply[1] == "-2" {
		log.Println("Failed to find application")
	}

HANGUP:
	snd_chan <- "HANGUP\n"
	reply = agi_response(<-rcv_chan)
	return
}

func agi_response(res string) []string {
	// Read back AGI repsonse
	res = strings.TrimRight(res, "\n")
	reply := strings.SplitN(res, " ", 3)

	if reply[0] == "200" {
		reply[1] = strings.TrimPrefix(reply[1], "result=")
		log.Println("AGI command returned:", reply)
	} else {
		log.Println("AGI command failed:", reply)
		reply = []string{"-1", "-1", "-1"}
	}
	return reply
}
