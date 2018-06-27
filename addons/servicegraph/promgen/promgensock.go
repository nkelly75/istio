// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package promgen generates service graphs from a prometheus backend.
package promgen

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"istio.io/istio/addons/servicegraph"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true;
	},
}

type promSockHandler struct {
	addr   string
	static *servicegraph.Static
	writer servicegraph.SerializeFn
}

// NewPromHandler returns a new http.Handler that will serve servicegraph data
// based on queries against a prometheus backend.
func NewPromSockHandler(addr string, static *servicegraph.Static, writer servicegraph.SerializeFn) http.Handler {
	return &promSockHandler{addr, static, writer}
}

func (p *promSockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Print("In promSockHandler")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		log.Print("About to close websocket")
		c.Close()
	}()

	ticker := time.NewTicker(time.Second * 1)
	// goroutine to write on the websocket, should be stopped if a close is detected
	go func() {
		for range ticker.C {
			log.Print("Ticker ...")

			// err = c.WriteMessage(websocket.TextMessage, []byte("NGK.."))
			// if err != nil {
			// 	log.Println("write:", err)
			// 	break
			// }
			w, err := c.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("NextWriter:", err)
				break
			}
			if _, err := io.WriteString(w, "NGK2.."); err != nil {
				log.Println("WriteString:", err)
				break
			}
			if err := w.Close(); err != nil {
				log.Println("Close:", err)
				break
			}
		}
	}()

	for {
		// We don't expect to read anything from the client on the websocket
		// but we do need to detect close of the socket so we can clean up
		if _, _, err := c.NextReader(); err != nil {
			log.Print("reader got error", err)
			c.Close()
			ticker.Stop()
			break;
		}
	}
}
