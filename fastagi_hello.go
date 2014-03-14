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
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
)

var (
	debug     = flag.Bool("debug", false, "Print debug information on stderr")
	listen    = flag.String("listen", "127.0.0.1", "Listening address")
	port      = flag.String("port", "4573", "Listening server port")
	listeners = flag.Int("runs", 4, "Pool size of Listeners")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	shutdown := false

	log.Printf("Starting FastAGI server on %v\n", net.JoinHostPort(*listen, *port))
	listener, err := net.Listen("tcp", net.JoinHostPort(*listen, *port))
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	wg := new(sync.WaitGroup)
	for i := 0; i < *listeners; i++ {
		go func() {
			for !shutdown {
				conn, err := listener.Accept()
				if err != nil {
					log.Println(err)
					continue
				}
				if *debug {
					log.Printf("Connected: %v <-> %v\n", conn.LocalAddr(), conn.RemoteAddr())
				}
				wg.Add(1)
				go agiConnHandle(conn, wg)
			}
		}()
	}
	signal := <-c
	log.Printf("Received %v, Waiting for remaining sessions to end and exit.\n", signal)
	shutdown = true
	wg.Wait()
}

func agiLogic(rcvChan <-chan string, sndChan chan<- string, agiArg map[string]string) {
	//Do AGI stuff
	reply := make([]string, 3)
	var file string
	defer func() {
		reply = nil
		close(sndChan)
	}()

	_, query := parseAgiReq(agiArg["request"])
	if query["file"] == nil {
		if *debug {
			log.Println("No arguments passed, exiting")
		}
		goto HANGUP
	}
	file = query["file"][0]
	//Check channel status and answer if not answered already
	sndChan <- "CHANNEL STATUS\n"
	reply = agiResponse(rcvChan, reply)
	if reply[0] != "200" {
		goto HANGUP
	} else if reply[1] != "6" {
		sndChan <- "ANSWER\n"
		reply = agiResponse(rcvChan, reply)
		if reply[0] != "200" {
			goto HANGUP
		} else if reply[1] == "-1" {
			log.Println("Failed to answer channel")
			goto HANGUP
		}
	}
	//Display message on console and playback a file
	sndChan <- "VERBOSE \"Paying back: " + file + "\" 0\n"
	reply = agiResponse(rcvChan, reply)
	if reply[0] != "200" {
		goto HANGUP
	}
	sndChan <- "STREAM FILE " + file + "  \"\"\n"
	reply = agiResponse(rcvChan, reply)
	if reply[0] != "200" {
		goto HANGUP
	} else if reply[1] == "-1" {
		log.Println("Failed to playback file", file)
	}

HANGUP:
	if *debug {
		log.Println("Hanging up.")
	}
	sndChan <- "HANGUP\n"
	reply = agiResponse(rcvChan, reply)
	return
}

func agiConnHandle(client net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
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
	i := 0
	for agiStr := range rcvChan {
		if agiStr == "" || i > 150 {
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
		i++
	}
	if *debug {
		log.Println("Finished reading AGI vars:")
		for key, value := range agiInput {
			log.Println(key + "\t\t" + value)
		}
	}
	return
}

func agiResponse(rcvChan <-chan string, reply []string) []string {
	//Parse and return AGI responce
	reply = []string{"", "", ""}
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
			reply = []string{"ERR", "", ""}
			break
		}
	}
	if *debug {
		log.Println("AGI command returned:", reply)
	}
	return reply
}

func parseAgiReq(request string) (string, url.Values) {
	//Parse AGI reguest return path and query params
	req, _ := url.Parse(request)
	query, _ := url.ParseQuery(req.RawQuery)
	return req.Path, query
}
