// FastAGI server example in Go
//
// Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
//
// This program is free software, distributed under the terms of
// the GNU General Public License Version 2. See the LICENSE file
// at the top of the source tree.


package main

import (
	"bufio"
	"log"
	"net"

	"github.com/zaf/agi"
)

func main() {
	ln, err := net.Listen("tcp", ":4573")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go connHandle(conn)
	}
}

func connHandle(c net.Conn) {
	defer c.Close()
	var rep agi.Reply
	var file string
	//Create a new FastAGI session
	myAgi := agi.New()
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	err := myAgi.Init(rw)
	if err != nil {
		log.Printf("Error Parsing AGI environment: %v\n", err)
		return
	}
	// Check passed arguments
	if myAgi.Env["arg_1"] == "" {
		log.Println("No arguments passed, exiting...")
		goto HANGUP
	}
	file = myAgi.Env["arg_1"]
	// Chech channel status
	rep, err = myAgi.ChannelStatus()
	if err != nil {
		log.Printf("AGI reply error: %v\n", err)
		return
	}
	//Answer channel if not already answered
	if rep.Res != 6 {
		rep, err = myAgi.Answer()
		if err != nil || rep.Res == -1 {
			log.Printf("Failed to answer channel: %v\n", err)
			return
		}
	}
	// Playback file
	rep, err = myAgi.StreamFile(file, "1234567890#*")
	if err != nil {
		log.Printf("AGI reply error: %v\n", err)
		return
	}
	if rep.Res == -1 {
		log.Printf("Failed to playback file: %s\n", file)
	}
HANGUP:
	//Hangup
	myAgi.Hangup()
	return
}
