// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"zn/client"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	log.Println("App started")

	hub := client.NewHub()
	go hub.Run()

	ctx, cancel := context.WithCancel(context.Background())

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		client.ServeWs(ctx, hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	cancel()
	log.Println("App stopped")
}
