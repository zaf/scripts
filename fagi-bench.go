/*
	A FastAGI benchmark in go

	Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

	This program is free software, distributed under the terms of
	the GNU General Public License Version 2. See the LICENSE file
	at the top of the source tree.
*/

package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	DEBUG = true
	PORT  = 4573
	RUNS  = 10000
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("Usage: ", os.Args[0], "host")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	host := os.Args[1]
	var wg sync.WaitGroup
	wg.Add(RUNS)

	//Spawn Connections to AGI server
	for i := 0; i < RUNS; i++ {
		go func() {
			defer wg.Done()
			conn, err := net.Dial("tcp", host+":"+strconv.Itoa(PORT))
			if err != nil {
				log.Println(err)
				return
			}
			init_data := agi_init()
			for key, value := range init_data {
				fmt.Fprintf(conn, key+": "+value+"\n")
			}
			fmt.Fprintf(conn, "\n")
			bufio.NewReader(conn).ReadString('\n')
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintf(conn, "200 result=0\n")
			bufio.NewReader(conn).ReadString('\n')
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintf(conn, "200 result=0\n")
			time.Sleep(500 * time.Millisecond)
			fmt.Fprintf(conn, "HANGUP\n")
			conn.Close()
			return
		}()
		time.Sleep(5 * time.Millisecond)
	}
	wg.Wait()
	os.Exit(0)
}

func agi_init() map[string]string {
	//Generate AGI initialisation data
	agi_data := map[string]string{
		"agi_network":        "yes",
		"agi_network_script": "bench",
		"agi_request":        "agi://" + os.Args[1],
		"agi_channel":        "ALSA/default",
		"agi_language":       "en",
		"agi_type":           "Console",
		"agi_uniqueid":       get_rand_str(),
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
		"agi_threadid":       get_rand_str(),
		"agi_arg_1":          "echo-test",
	}
	return agi_data
}

func get_rand_str() string {
	//Generate a 9 digit random numeric string
	return strconv.Itoa(100000000 + rand.Intn(899999999))
}
