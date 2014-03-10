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
	"flag"
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

//benchmark parameters and default values
var (
	shutdown = false
	file     *os.File
	writer   *bufio.Writer
	debug    = flag.Bool("debug", false, "Write detailed statistics output to csv file")
	host     = flag.String("host", "127.0.0.1", "FAstAGI server host")
	port     = flag.String("port", "4573", "FastAGI server port")
	runs     = flag.Int("runs", 10, "Number of runs per second")
	sess     = flag.Int("sess", 10, "Sessions per run")
	delay    = flag.Int("delay", 100, "Delay in AGI responses to the server (milliseconds)")
	req      = flag.String("req", "echo_test?par=foo", "AGI request")
	arg      = flag.String("arg", "foo", "Argument to pass to the FastAGI server")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *debug {
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
		writer.WriteString("#Started benchmark at: " + time.Now().String())
		writer.WriteString("\n#Host: " + *host + "\n#Port: " + *port + "\n#Runs: " + strconv.Itoa(*runs))
		writer.WriteString("\n#Sessions: " + strconv.Itoa(*sess) + "\n#Delay: " + strconv.Itoa(*delay) + "\n#Reguest: " + *req)
		writer.WriteString("\n#completed,active,duration\n")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	wg := new(sync.WaitGroup)
	wg.Add(1)
	//Start benchmark and wait for users input to stop
	go agiBench(wg)
	bufio.NewReader(os.Stdin).ReadString('\n')
	shutdown = true
	wg.Wait()
	if *debug {
		writer.WriteString("#Stopped benchmark at: " + time.Now().String() + "\n")
	}
}

func agiBench(wg *sync.WaitGroup) {
	defer wg.Done()
	var active, count, fail int32
	var avrDur int64
	logChan := make(chan string, *sess*2)
	timeChan := make(chan int64, *sess*2)
	runDelay := time.Duration(1000000000 / *runs) * time.Nanosecond
	replyDelay := time.Duration(*delay) * time.Millisecond
	wg1 := new(sync.WaitGroup)
	wg1.Add(*sess)
	//Spawn pool of paraller runs
	for i := 0; i < *sess; i++ {
		ticker := time.Tick(runDelay)
		go func(ticker <-chan time.Time) {
			defer wg1.Done()
			wg2 := new(sync.WaitGroup)
			for !shutdown {
				<-ticker
				wg2.Add(1)
				//Spawn Connections to the AGI server
				go func() {
					defer wg2.Done()
					initData := agiInit()
					start := time.Now()
					conn, err := net.Dial("tcp", *host+":"+*port)
					if err != nil {
						atomic.AddInt32(&fail, 1)
						if *debug {
							logChan <- fmt.Sprintf("# %s\n", err)
						}
						return
					}
					atomic.AddInt32(&active, 1)
					scanner := bufio.NewScanner(conn)
					//Send AGI initialisation data
					for key, value := range initData {
						conn.Write([]byte(key + ": " + value + "\n"))
					}
					conn.Write([]byte("\n"))
					//Reply with '200' to all messages from the AGI server until it hangs up
					for scanner.Scan() {
						time.Sleep(replyDelay)
						conn.Write([]byte("200 result=0\n"))
					}
					conn.Close()
					elapsed := time.Since(start)
					timeChan <- elapsed.Nanoseconds()
					atomic.AddInt32(&active, -1)
					atomic.AddInt32(&count, 1)
					if *debug {
						logChan <- fmt.Sprintf("%d,%d,%d\n", atomic.LoadInt32(&count), atomic.LoadInt32(&active), elapsed.Nanoseconds())
					}
				}()

			}
			wg2.Wait()
		}(ticker)
	}
	wg3 := new(sync.WaitGroup)
	//Write to log file
	if *debug {
		wg3.Add(1)
		go func() {
			defer wg3.Done()
			for logMsg := range logChan {
				writer.WriteString(logMsg)
			}
		}()
	}
	//Calculate Average session duration for the last 1000 sessions
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		var sessions int64
		for dur := range timeChan {
			sessions++
			avrDur = (avrDur*(sessions-1) + dur) / sessions
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
				runDelay, "\nSessions per run:", *sess, "\nReply delay:", replyDelay,
				"\n\nFastAGI Sessions\nActive:", atomic.LoadInt32(&active), "\nCompleted:",
				atomic.LoadInt32(&count), "\nDuration:", atomic.LoadInt64(&avrDur),
				"ns (last 1000 sessions average)\nFailed:", atomic.LoadInt32(&fail))
			if shutdown {
				fmt.Println("Stopping...")
				if atomic.LoadInt32(&active) == 0 {
					break
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	//Wait all FastAGI sessions to end
	wg1.Wait()
	close(logChan)
	close(timeChan)
	//Wait writing to log file to finish
	wg3.Wait()
}

func agiInit() map[string]string {
	//Generate AGI initialisation data
	agiData := map[string]string{
		"agi_network":        "yes",
		"agi_network_script": *req,
		"agi_request":        "agi://" + *host + "/" + *req,
		"agi_channel":        "SIP/1234-00000000",
		"agi_language":       "en",
		"agi_type":           "SIP",
		"agi_uniqueid":       strconv.Itoa(100000000 + rand.Intn(899999999)),
		"agi_version":        "0.1",
		"agi_callerid":       "1234",
		"agi_calleridname":   "1234",
		"agi_callingpres":    "67",
		"agi_callingani2":    "0",
		"agi_callington":     "0",
		"agi_callingtns":     "0",
		"agi_dnid":           "100",
		"agi_rdnis":          "unknown",
		"agi_context":        "default",
		"agi_extension":      "100",
		"agi_priority":       "1",
		"agi_enhanced":       "0.0",
		"agi_accountcode":    "",
		"agi_threadid":       strconv.Itoa(100000000 + rand.Intn(899999999)),
		"agi_arg_1":          *arg,
	}
	return agiData
}
