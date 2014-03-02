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
	"time"
)

const (
	DEBUG = true
	PORT  = 4573
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("Usage: ", os.Args[0], "host")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	host := os.Args[1]
	for i := 0; i < 10000; i++ {
		go func(i int) {
			conn, err := net.Dial("tcp", host+":"+strconv.Itoa(PORT))
			if err != nil {
				log.Println(err)
				return
			}

			init_data := agi_init()
			//log.Print("Starting connection:", strconv.Itoa(i))
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
			//log.Print("Closing connection:", strconv.Itoa(i))
			conn.Close()
		}(i)
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(5 * time.Second)
	os.Exit(0)
}

func agi_init() map[string]string {
	agi_data := make(map[string]string)
	agi_data["agi_network"] = "yes"
	agi_data["agi_network_script"] = "bench"
	agi_data["agi_request"] = "agi://" + os.Args[1]
	agi_data["agi_channel"] = "ALSA/default"
	agi_data["agi_language"] = "en"
	agi_data["agi_type"] = "Console"
	agi_data["agi_uniqueid"] = get_rand_str()
	agi_data["agi_version"] = "0.1"
	agi_data["agi_callerid"] = "unknown"
	agi_data["agi_calleridname"] = "unknown"
	agi_data["agi_callingpres"] = "67"
	agi_data["agi_callingani2"] = "0"
	agi_data["agi_callington"] = "0"
	agi_data["agi_callingtns"] = "0"
	agi_data["agi_dnid"] = "unknown"
	agi_data["agi_rdnis"] = "unknown"
	agi_data["agi_context"] = "default"
	agi_data["agi_extension"] = "100"
	agi_data["agi_priority"] = "1"
	agi_data["agi_enhanced"] = "0.0"
	agi_data["agi_accountcode"] = ""
	agi_data["agi_threadid"] = get_rand_str()
	agi_data["agi_arg_1"] = "echo-test"
	return agi_data
}

func get_rand_str() string {
	return strconv.Itoa(10000000 + rand.Intn(89999999))
}
