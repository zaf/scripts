/*
	A FastAGI paraller benchmark in go

	Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

	This program is free software, distributed under the terms of
	the GNU General Public License Version 2. See the LICENSE file
	at the top of the source tree.
*/

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//General benchmark parameters
const (
	PORT     = 4573        //FastAGI server port
	RUNS_SEC = 10          //Number of runs per second
	SESS_RUN = 2           //Sessions per run
	SESS_DUR = 2           //Session duration in sec
	AGI_ARG1 = "echo-test" //Argument to pass to the FastAGI server
)

var shutdown bool = false
var last_error error

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "host")
		os.Exit(1)
	}
	rand.Seed(time.Now().UTC().UnixNano())
	ch1 := make(chan bool)
	fmt.Print("\033[2J\033[H")
	go agi_session(os.Args[1], ch1)
	bufio.NewReader(os.Stdin).ReadString('\n')
	shutdown = true
	fmt.Println("Stopping...")
	<-ch1
	os.Exit(0)
}

func agi_session(host string, c chan<- bool) {
	//Spawn Connections to AGI server
	count := 0
	fail := 0
	delay := time.Duration(1000/RUNS_SEC/SESS_RUN) * time.Millisecond
	half_duration := time.Duration(1000*SESS_DUR/2) * time.Millisecond
	ticker := time.Tick(delay)
	wg := sync.WaitGroup{}
	wg.Add(SESS_RUN)
	for i := 0; i < SESS_RUN; i++ {
		go func(ticker <-chan time.Time) {
			for !shutdown {
				go func() {
					conn, err := net.Dial("tcp", host+":"+strconv.Itoa(PORT))
					if err != nil {
						fail++
						last_error = err
						return
					}
					count++
					init_data := agi_init(host)
					for key, value := range init_data {
						fmt.Fprintf(conn, key+": "+value+"\n")
					}
					fmt.Fprintf(conn, "\n")
					bufio.NewReader(conn).ReadString('\n')
					time.Sleep(half_duration)
					conn.Write([]byte("200 result=0\n"))
					bufio.NewReader(conn).ReadString('\n')
					time.Sleep(half_duration)
					conn.Write([]byte("HANGUP\n"))
					conn.Close()
					count--
					return
				}()
				<-ticker
			}
			wg.Done()
		}(ticker)
	}
	go func() {
		for !shutdown {
			fmt.Println("Running paraller AGI bench:\nPress Enter to stop.\n\nA new run each:  ",
				delay, "\nSessions per run:", SESS_RUN, "\nSession duration:", 2*half_duration)
			fmt.Println("\nActive Sessions:", count, "\nFailed sessions:", fail)
			if last_error != nil {
				fmt.Println("Last error:", last_error)
			}
			time.Sleep(500 * time.Millisecond)
			fmt.Print("\033[2J\033[H")
		}
	}()
	wg.Wait()
	c <- true
	return
}

func agi_init(host string) map[string]string {
	//Generate AGI initialisation data
	agi_data := map[string]string{
		"agi_network":        "yes",
		"agi_network_script": "bench",
		"agi_request":        "agi://" + host,
		"agi_channel":        "ALSA/default",
		"agi_language":       "en",
		"agi_type":           "Console",
		"agi_uniqueid":       strconv.Itoa(100000000 + rand.Intn(899999999)),
		"agi_version":        "0.1",
		"agi_callerid":       "unknown",
		"agi_calleridname":   "unknown",
		"agi_callingpres":    "67",
		"agi_callingani2":    "0",
		"agi_callington":     "0",
		"agi_callingtns":     "0",
		"agi_dnid":           "unknown",
		"agi_rdnis":          "unknown",
		"agi_context":        "default",
		"agi_extension":      "100",
		"agi_priority":       "1",
		"agi_enhanced":       "0.0",
		"agi_accountcode":    "",
		"agi_threadid":       strconv.Itoa(100000000 + rand.Intn(899999999)),
		"agi_arg_1":          AGI_ARG1,
	}
	return agi_data
}
