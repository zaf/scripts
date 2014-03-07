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
	"sync/atomic"
	"time"
)

//General benchmark parameters
const (
	DEBUG    = true        //Enable detailed statistics output to file bench.csv
	PORT     = 4573        //FastAGI server port
	RUNS_SEC = 10          //Number of runs per second
	SESS_RUN = 10          //Sessions per run
	DELAY    = 100         //Delay in AGI responses to the server (milliseconds)
	AGI_ARG1 = "echo-test" //Argument to pass to the FastAGI server
)

var (
	shutdown bool = false
	file     *os.File
	writer   *bufio.Writer
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "host")
		os.Exit(1)
	}
	if DEBUG {
		//Open log file for writing
		file, err := os.Create("bench-" + strconv.FormatInt(time.Now().Unix(), 10) + ".csv")
		if err != nil {
			fmt.Println("Failed to create file:", err)
			os.Exit(1)
		}
		writer = bufio.NewWriter(file)
		defer func() {
			writer.Flush()
			file.Close()
		}()
		writer.WriteString("#Starting benchmark at: " + time.Now().String() + "\n#completed,active,duration\n")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	wg := new(sync.WaitGroup)
	wg.Add(1)
	//Start benchmark and wait for users input to stop
	go agi_bench(os.Args[1], wg)
	bufio.NewReader(os.Stdin).ReadString('\n')
	shutdown = true
	wg.Wait()
	if DEBUG {
		writer.WriteString("#Stopped benchmark at: " + time.Now().String())
	}
}

func agi_bench(host string, wg *sync.WaitGroup) {
	defer wg.Done()
	var active, count, fail int32 = 0, 0, 0
	var avg_dur int64 = 0
	log_chan := make(chan string, SESS_RUN*2)
	time_chan := make(chan int64, SESS_RUN*2)
	run_delay := time.Duration(1000000000/RUNS_SEC) * time.Nanosecond
	reply_delay := time.Duration(DELAY) * time.Millisecond
	wg1 := new(sync.WaitGroup)
	wg1.Add(SESS_RUN)
	//Spawn pool of paraller runs
	for i := 0; i < SESS_RUN; i++ {
		ticker := time.Tick(run_delay)
		go func(ticker <-chan time.Time) {
			defer wg1.Done()
			wg2 := new(sync.WaitGroup)
			for !shutdown {
				<-ticker
				wg2.Add(1)
				//Spawn Connections to the AGI server
				go func() {
					defer wg2.Done()
					conn, err := net.Dial("tcp", host+":"+strconv.Itoa(PORT))
					if err != nil {
						atomic.AddInt32(&fail, 1)
						if DEBUG {
							log_chan <- fmt.Sprintf("# %s\n", err)
						}
						return
					}
					atomic.AddInt32(&active, 1)
					scanner := bufio.NewScanner(conn)
					init_data := agi_init(host)
					start := time.Now()
					//Send AGI initialisation data
					for key, value := range init_data {
						conn.Write([]byte(key + ": " + value + "\n"))
					}
					conn.Write([]byte("\n"))
					//Reply with '200' to all messages from the AGI server until it hangs up
					for scanner.Scan() {
						time.Sleep(reply_delay)
						conn.Write([]byte("200 result=0\n"))
					}
					conn.Close()
					elapsed := time.Since(start)
					time_chan <- elapsed.Nanoseconds()
					atomic.AddInt32(&active, -1)
					atomic.AddInt32(&count, 1)
					if DEBUG {
						log_chan <- fmt.Sprintf("%d,%d,%d\n", atomic.LoadInt32(&count), atomic.LoadInt32(&active), elapsed.Nanoseconds())
					}
				}()

			}
			wg2.Wait()
		}(ticker)
	}
	wg3 := new(sync.WaitGroup)
	//Write to log file
	if DEBUG {
		wg3.Add(1)
		go func() {
			defer wg3.Done()
			for log_msg := range log_chan {
				writer.WriteString(log_msg)
			}
		}()
	}
	//Calculate Average session duration for the last 1000 sessions
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		var sessions int64 = 0
		for dur := range time_chan {
			sessions++
			avg_dur = (avg_dur*(sessions-1) + dur) / sessions
			if sessions >= 1000 {
				sessions = 0
			}
		}
	}()
	//Display pretty output to the user
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		for {
			fmt.Print("\033[2J\033[H") //Clear screen
			fmt.Println("Running paraller AGI bench:\nPress Enter to stop.\n\nA new run each:",
				run_delay, "\nSessions per run:", SESS_RUN, "\nReply delay:", reply_delay,
				"\n\nFastAGI Sessions\nActive:", atomic.LoadInt32(&active), "\nCompleted:",
				atomic.LoadInt32(&count), "\nDuration:", atomic.LoadInt64(&avg_dur),
				"ns (last 1000 sessions average)\nFailed:", atomic.LoadInt32(&fail))
			if shutdown {
				fmt.Println("Stopping...")
				if atomic.LoadInt32(&active) == 0 {
					break
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	//Wait all FastAGI sessions to end
	wg1.Wait()
	close(log_chan)
	close(time_chan)
	//Wait writing to log file to finish
	wg3.Wait()
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
