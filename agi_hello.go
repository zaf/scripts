/*	A simple AGI example in go
	We read and store AGI input, run some simple AGI commands and parse the output

	Copyright (C) 2013, Lefteris Zafiris <zaf.000@gmail.com>

	This program is free software, distributed under the terms of
	the GNU General Public License Version 2. See the LICENSE file
	at the top of the source tree.
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var debug = true
var agi_reader = bufio.NewReaderSize(os.Stdin, 0)

func main() {
	var res []string
	agi_data := make(map[string]string)

	agi_init(agi_data)
	if agi_data["arg_1"] == "" {
		fmt.Fprintln(os.Stderr, "No arguments passed, exiting")
		os.Exit(1)
	}
	my_file := agi_data["arg_1"]

	//Check channel status
	fmt.Fprintln(os.Stdout, "CHANNEL STATUS")
	res = agi_response()
	//Answer channel if not answered already
	if res[1] == "4" {
		fmt.Fprintln(os.Stdout, "ANSWER")
		res = agi_response()
		if res[1] == "-1" {
			fmt.Fprintln(os.Stderr, "Failed to answer channel")
			os.Exit(1)
		}
	}
	//Display on the console the file we are about to playback
	fmt.Fprintln(os.Stdout, "VERBOSE \"Playingback file:", my_file, "\" 1")
	//os.Stdout.Sync()
	res = agi_response()
	//Playback file
	fmt.Fprintln(os.Stdout, "STREAM FILE", my_file, "\"\"")
	//os.Stdout.Sync()
	res = agi_response()
	os.Exit(0)
}

func agi_init(agi_arg map[string]string) {
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
		fmt.Fprintln(os.Stderr, "Finished reading AGI vars:")
		for key, value := range agi_arg {
			fmt.Fprintln(os.Stderr, key+"\t\t"+value)
		}
	}
}

func agi_response() []string {
	// Read back AGI repsonse
	line, _ := agi_reader.ReadString('\n')
	res := strings.Replace(line, "\n", "", -1)
	reply := strings.SplitN(res, " ", 3)

	if len(reply) < 2 {
		fmt.Fprintln(os.Stderr, "AGI unexpected error!")
		return []string{"-1", "-1", "-1"}
	}
	if reply[0] != "200" {
		fmt.Fprintln(os.Stderr, "AGI command failed:", reply)
	} else {
		reply[1] = strings.TrimPrefix(reply[1], "result=")
	}
	if debug {
		fmt.Fprintln(os.Stderr, "AGI command returned:", reply)
	}
	return reply
}
