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
	"flag"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
)

var (
	debug     = flag.Bool("debug", false, "Print debug information on stderr")
	host      = flag.String("host", "127.0.0.1", "Listening address")
	port      = flag.String("port", "4573", "Listening server port")
	listeners = flag.Int("runs", 5, "Pool size of Listeners")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.Println("Starting FastAGI server...")

	listener, err := net.Listen("tcp", *host+":"+*port)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	wg := new(sync.WaitGroup)
	wg.Add(*listeners)
	for i := 0; i < *listeners; i++ {
		go func() {
			defer wg.Done()
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Println(err)
					continue
				}
				if *debug {
					log.Printf("Connected: %v <-> %v\n", conn.LocalAddr(), conn.RemoteAddr())
				}
				go agiConnHandle(conn)
			}
		}()
	}
	wg.Wait()
}

func agiLogic(rcvChan <-chan string, sndChan chan<- string, agiArg map[string]string) {
	//Do AGI stuff
	reply := make([]string, 3)
	defer func() {
		reply = nil
		close(sndChan)
	}()

	if agiArg["arg_1"] == "" {
		if *debug {
			log.Println("No arguments passed, exiting")
		}
		goto HANGUP
	}

	sndChan <- "VERBOSE \"Staring an echo test.\" 3\n"
	reply = agiResponse(rcvChan)
	if reply[0] != "200" {
		goto HANGUP
	}

	//Check channel status and answer if not answered already
	sndChan <- "CHANNEL STATUS\n"
	reply = agiResponse(rcvChan)
	if reply[0] != "200" {
		goto HANGUP
	} else if reply[1] != "6" {
		sndChan <- "ANSWER\n"
		reply = agiResponse(rcvChan)
		if reply[0] != "200" {
			goto HANGUP
		} else if reply[1] == "-1" {
			log.Println("Failed to answer channel")
			goto HANGUP
		}
	}
	//Playback a file and run the echo() app
	sndChan <- "STREAM FILE " + agiArg["arg_1"] + "  \"\"\n"
	reply = agiResponse(rcvChan)
	if reply[0] != "200" {
		goto HANGUP
	} else if reply[1] == "-1" {
		log.Println("Failed to playback file", agiArg["arg_1"])
	}
	sndChan <- "EXEC Echo\n"
	reply = agiResponse(rcvChan)
	if reply[0] != "200" {
		goto HANGUP
	} else if reply[1] == "-2" {
		log.Println("Failed to find application")
	}

HANGUP:
	sndChan <- "HANGUP\n"
	reply = agiResponse(rcvChan)
	return
}

func agiConnHandle(client net.Conn) {
	rcvChan := make(chan string)
	sndChan := make(chan string)
	agiData := make(map[string]string)
	//Receive network data and send to channel
	go func() {
		scanner := bufio.NewScanner(client)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "HANGUP" {
				if *debug {
					log.Println("Client hung up.")
				}
				break
			}
			rcvChan <- line
		}
		if *debug {
			log.Printf("Connection from %v closed.", client.RemoteAddr())
		}
		client.Close()
		close(rcvChan)
		return
	}()
	//Read channel data and send to network
	go func() {
		for {
			select {
			case agiMsg, ok := <-sndChan:
				if !ok {
					if *debug {
						log.Printf("Channel closed.")
					}
					return
				}
				_, err := client.Write([]byte(agiMsg))
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}()

	agiInit(rcvChan, agiData)
	agiLogic(rcvChan, sndChan, agiData)

	client.Close()
	agiData = nil
	return
}

func agiInit(rcvChan <-chan string, agiInput map[string]string) {
	//Read and store AGI input
	for agiStr := range rcvChan {
		if agiStr == "" {
			break
		}
		inputStr := strings.SplitN(agiStr, ": ", 2)
		if len(inputStr) == 2 {
			inputStr[0] = strings.TrimPrefix(inputStr[0], "agi_")
			inputStr[1] = strings.TrimRight(inputStr[1], "\n")
			agiInput[inputStr[0]] = inputStr[1]
		} else {
			log.Println("No AGI Compatible Input:", inputStr)
			break
		}
	}
	if *debug {
		log.Println("Finished reading AGI vars:")
		for key, value := range agiInput {
			log.Println(key + "\t\t" + value)
		}
	}
	return
}

func agiResponse(rcvChan <-chan string) []string {
	//Parse and return AGI responce
	reply := make([]string, 3)
	for msg := range rcvChan {
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
			if *debug {
				log.Println("AGI unexpected response:", reply)
			}
			return []string{"ERR", "", ""}
		}
	}
	if *debug {
		log.Println("AGI command returned:", reply)
	}
	return reply
}
