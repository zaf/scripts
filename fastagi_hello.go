/*
	A concurrent FastAGI server example in go

	Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

	This program is free software, distributed under the terms of
	the GNU General Public License Version 2. See the LICENSE file
	at the top of the source tree.
*/

package main

import (
	"bufio"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	DEBUG     = true      //Print debug information on stderr
	PORT      = 4573      //Listening port
	HOST      = "0.0.0.0" //Listening address
	LISTENERS = 5         //Number of Listeners
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("Starting FastAGI server...")

	listener, err := net.Listen("tcp", HOST+":"+strconv.Itoa(PORT))
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	wg := sync.WaitGroup{}
	wg.Add(LISTENERS)
	for i := 0; i < LISTENERS; i++ {
		go func() {
			defer wg.Done()
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Println(err)
					continue
				}
				if DEBUG {
					log.Printf("Connected: %v <-> %v\n", conn.LocalAddr(), conn.RemoteAddr())
				}
				go agi_conn_handle(conn)
			}
		}()
	}
	wg.Wait()
}

func agi_logic(rcv_chan <-chan string, snd_chan chan<- string, agi_arg map[string]string) {
	//Do AGI stuff
	reply := make([]string, 3)
	if agi_arg["arg_1"] == "" {
		if DEBUG {
			log.Println("No arguments passed, exiting")
		}
		goto END
	}

	snd_chan <- "VERBOSE \"Staring an echo test.\" 3\n"
	reply = agi_response(rcv_chan)
	if reply[0] != "200" {
		goto END
	}

	//Check channel status and answer if not answered already
	snd_chan <- "CHANNEL STATUS\n"
	reply = agi_response(rcv_chan)
	if reply[0] != "200" {
		goto END
	} else if reply[1] != "6" {
		snd_chan <- "ANSWER\n"
		reply = agi_response(rcv_chan)
		if reply[0] != "200" {
			goto END
		} else if reply[1] == "-1" {
			log.Println("Failed to answer channel")
			goto HANGUP
		}
	}
	//Playback a file and run the echo() app
	snd_chan <- "STREAM FILE " + agi_arg["arg_1"] + "  \"\"\n"
	reply = agi_response(rcv_chan)
	if reply[0] != "200" {
		goto END
	} else if reply[1] == "-1" {
		log.Println("Failed to playback file", agi_arg["arg_1"])
	}
	snd_chan <- "EXEC Echo\n"
	reply = agi_response(rcv_chan)
	if reply[0] != "200" {
		goto END
	} else if reply[1] == "-2" {
		log.Println("Failed to find application")
	}

HANGUP:
	snd_chan <- "HANGUP\n"
	reply = agi_response(rcv_chan)
END:
	reply = nil
	close(snd_chan)
	return
}

func agi_conn_handle(client net.Conn) {
	rcv_chan := make(chan string)
	snd_chan := make(chan string)
	agi_data := make(map[string]string)
	//Receive network data and send to channel
	go func() {
		scanner := bufio.NewScanner(client)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "HANGUP" {
				if DEBUG {
					log.Println("Client hung up.")
				}
				break
			}
			rcv_chan <- line
		}
		if DEBUG {
			log.Printf("Connection from %v closed.", client.RemoteAddr())
		}
		client.Close()
		close(rcv_chan)
		return
	}()
	//Read channel data and send to network
	go func() {
		for {
			select {
			case agi_msg, ok := <-snd_chan:
				if !ok {
					if DEBUG {
						log.Printf("Channel closed.")
					}
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
	}()

	agi_init(rcv_chan, agi_data)
	agi_logic(rcv_chan, snd_chan, agi_data)

	client.Close()
	agi_data = nil
	return
}

func agi_init(rcv_chan <-chan string, agi_input map[string]string) {
	//Read and store AGI input
	for agi_str := range rcv_chan {
		if agi_str == "" {
			break
		}
		input_str := strings.SplitN(agi_str, ": ", 2)
		if len(input_str) == 2 {
			input_str[0] = strings.TrimPrefix(input_str[0], "agi_")
			input_str[1] = strings.TrimRight(input_str[1], "\n")
			agi_input[input_str[0]] = input_str[1]
		} else {
			log.Println("No AGI Compatible Input:", input_str)
			break
		}
	}
	if DEBUG {
		log.Println("Finished reading AGI vars:")
		for key, value := range agi_input {
			log.Println(key + "\t\t" + value)
		}
	}
	return
}

func agi_response(rcv_chan <-chan string) []string {
	//Parse and return AGI responce
	reply := make([]string, 3)
	for msg := range rcv_chan {
		msg = strings.TrimRight(msg, "\n\r")
		if reply[0] == "520" {
			break
		}
		reply = strings.SplitN(msg, " ", 3)
		if reply[0] == "200" {
			reply[1] = strings.TrimPrefix(reply[1], "result=")
			break
		} else if reply[0] == "510" {
			reply[1] = "Invalid or unknown command."
			reply[2] = ""
			break
		} else if reply[0] == "511" {
			reply[1] = "Command Not Permitted on a dead channel."
			reply[2] = ""
			break
		} else if reply[0] == "520" {
			reply[0] = "520"
			reply[1] = "Invalid command syntax."
			reply[2] = ""
			break
		} else if reply[0] == "520-Invalid" {
			reply[0] = "520"
			reply[1] = "Invalid command syntax."
			reply[2] = ""
		} else {
			if DEBUG {
				log.Println("AGI unexpected response:", reply)
			}
			return []string{"ERR", "", ""}
		}
	}
	if DEBUG {
		log.Println("AGI command returned:", reply)
	}
	return reply
}
