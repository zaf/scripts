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
	DEBUG    = false       //Enable detailed statistics output to file bench.csv
	PORT     = 4573        //FastAGI server port
	RUNS_SEC = 10          //Number of runs per second
	SESS_RUN = 2           //Sessions per run
	SESS_DUR = 5           //Session duration in sec
	AGI_ARG1 = "echo-test" //Argument to pass to the FastAGI server
)

var shutdown bool = false

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "host")
		os.Exit(1)
	}
// 	if DEBUG {
// 		//Open file for writing
// 		file, err := os.Create("bench.csv")
// 		if err != nil {
// 			fmt.Println("Failed to create file:", err)
// 			os.Exit(1)
// 		}
// 		writer := bufio.NewWriter(file)
// 		defer func() {
// 			writer.Flush()
// 			file.Close()
// 		}()
// 		writer.WriteString("Starting benchmark at: "+time.Now().String()+"\nrun,active,duration\n")
// 	}
	rand.Seed(time.Now().UTC().UnixNano())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go agi_session(os.Args[1], &wg)
	bufio.NewReader(os.Stdin).ReadString('\n')
	shutdown = true
	wg.Wait()
// 	if DEBUG {
// 		writer.WriteString("Stopped benchmark at: "+time.Now().String()+"\n")
// 	}
	os.Exit(0)
}

func agi_session(host string, wg *sync.WaitGroup) {
	//Spawn Connections to AGI server
	defer wg.Done()
	var last_error error
	active, count, fail := 0, 0, 0
	delay := time.Duration(1000/RUNS_SEC/SESS_RUN) * time.Millisecond
	half_duration := time.Duration(1000*SESS_DUR/2) * time.Millisecond
	ticker := time.Tick(delay)
	wg1 := sync.WaitGroup{}
	wg1.Add(SESS_RUN + 1)
	for i := 0; i < SESS_RUN; i++ {
		go func(ticker <-chan time.Time) {
			defer wg1.Done()
			wg2 := sync.WaitGroup{}
			for !shutdown {
				<-ticker
				wg2.Add(1)
				go func() {
					defer wg2.Done()
					//var start time.Time
					//var elapsed time.Duration
					conn, err := net.Dial("tcp", host+":"+strconv.Itoa(PORT))
					if err != nil {
						fail++
						last_error = err
						return
					}
					active++
					init_data := agi_init(host)
					//start = time.Now()
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
					//elapsed = time.Since(start)
					active--
					count++
					return
				}()

			}
			wg2.Wait()
		}(ticker)
	}
	go func() {
		defer wg1.Done()
		//var average time.Duration = 0
		for {
			fmt.Print("\033[2J\033[H")
			fmt.Println("Running paraller AGI bench:\nPress Enter to stop.\n\nA new run each:  ",
				delay*SESS_RUN, "\nSessions per run:", SESS_RUN, "\nSession duration:", 2*half_duration)
			fmt.Println("\nFastAGI Sessions\nActive:", active, "\nCompleted:", count, "\nFailed:", fail)
			//if count != 0 {
			//	average = (average*time.Duration(count-1) + elapsed) / time.Duration(count)
			//	fmt.Println("\nAverage sessions duration:", average)
			//}
			if last_error != nil {
				fmt.Println("Last error:", last_error)
			}
			if shutdown {
				fmt.Println("Stopping...")
				if active == 0 { break }
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	wg1.Wait()
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
