/*	A simple AGI example in go
	We read and store AGI input, run some simple AGI commands and parse the output

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
	"os"
	"strings"
)

var (
	debug      = true
	agi_reader = bufio.NewReaderSize(os.Stdin, 0)
)

func main() {
	var res []string
	agi_data := make(map[string]string)

	agi_parse_init(agi_data)
	if agi_data["arg_1"] == "" {
		log.Fatalln("No arguments passed, exiting")
	}
	my_file := agi_data["arg_1"]

	//Check channel status
	fmt.Fprintln(os.Stdout, "CHANNEL STATUS")
	res = agi_parse_response()
	if res[0] != "200" {
		log.Fatalln("Failed to get Channel status")
	}
	//Answer channel if not answered already
	if res[1] != "6" {
		fmt.Fprintln(os.Stdout, "ANSWER")
		res = agi_parse_response()
		if res[0] != "200" || res[1] == "-1" {
			log.Fatalln("Failed to answer channel")
		}
	}
	//Display on the console the file we are about to playback
	fmt.Fprintln(os.Stdout, "VERBOSE \"Playingback file: "+my_file+"\" 1")
	//os.Stdout.Sync()
	res = agi_parse_response()
	if res[0] != "200" {
		log.Fatalln("VERBOSE failed")
	}
	//Playback file
	fmt.Fprintln(os.Stdout, "STREAM FILE", my_file, "\"\"")
	//os.Stdout.Sync()
	res = agi_parse_response()
	if res[0] != "200" || res[1] == "-1" {
		log.Fatalln("Failed to playback file", my_file)
	}
	os.Exit(0)
}

func agi_parse_init(agi_arg map[string]string) {
	//Read and store AGI input
	for {
		line, err := agi_reader.ReadString('\n')
		if err != nil || line == "\n" {
			break
		}
		input_str := strings.SplitN(line, ": ", 2)
		if len(input_str) == 2 {
			input_str[0] = strings.TrimPrefix(input_str[0], "agi_")
			input_str[1] = strings.TrimRight(input_str[1], "\n")
			agi_arg[input_str[0]] = input_str[1]
		}
	}

	if debug {
		log.Println("Finished reading AGI vars:")
		for key, value := range agi_arg {
			log.Println(key + "\t\t" + value)
		}
	}
}

func agi_parse_response() []string {
	// Read back AGI repsonse
	reply := make([]string, 3)
	line, _ := agi_reader.ReadString('\n')
	line = strings.TrimRight(line, "\n")
	reply = strings.SplitN(line, " ", 3)

	if reply[0] == "200" {
		reply[1] = strings.TrimPrefix(reply[1], "result=")
	} else if reply[0] == "510" {
		reply[1] = "Invalid or unknown command."
		reply[2] = ""
	} else if reply[0] == "511" {
		reply[1] = "Command Not Permitted on a dead channel."
		reply[2] = ""
	} else if reply[0] == "520" {
		reply[0] = "520"
		reply[1] = "Invalid command syntax."
		reply[2] = ""
	} else if reply[0] == "520-Invalid" {
		reply[0] = "520"
		reply[1] = "Invalid command syntax."
		reply[2], _ = agi_reader.ReadString('\n')
		reply[2] = "Proper usage follows: " + strings.TrimRight(reply[2], "\n")
	} else {
		log.Println("AGI unexpected response:", reply)
		return []string{"ERR", "", ""}
	}

	if debug {
		log.Println("AGI command returned:", reply)
	}
	return reply
}
