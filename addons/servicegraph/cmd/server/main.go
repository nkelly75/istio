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

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"

	"istio.io/istio/addons/servicegraph"
	"istio.io/istio/addons/servicegraph/dot"
	"istio.io/istio/addons/servicegraph/promgen"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true;
	},
}

func writeJSON(w io.Writer, g *servicegraph.Dynamic) error {
	return json.NewEncoder(w).Encode(g)
}

type justFilesFilesystem struct {
	Fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, _ := f.Stat()
	if stat.IsDir() {
		return nil, os.ErrNotExist
	}
	return f, nil
}

func (s *state) addNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotImplemented)
		_, err := w.Write([]byte("requests of this type not supported at this time"))
		if err != nil {
			log.Print(err)
		}
		return
	}
	nodeName := r.URL.Query().Get("name")
	if nodeName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("missing argument 'name'"))
		if err != nil {
			log.Print(err)
		}
		return
	}
	s.staticGraph.Nodes[nodeName] = struct{}{}
}

type state struct {
	staticGraph *servicegraph.Static
}

func vizSocket(w http.ResponseWriter, r *http.Request) {
	log.Print("Entered vizSocket")

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
			log.Print("Ticker ...");
			err = c.WriteMessage(websocket.TextMessage, []byte("NGK.."))
			if err != nil {
				log.Println("write:", err)
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

func main() {
	bindAddr := flag.String("bindAddr", ":8088", "Address to bind to for serving")
	promAddr := flag.String("prometheusAddr", "http://localhost:9090", "Address of prometheus instance for graph generation")
	assetDir := flag.String("assetDir", "./", "directory find assets to serve")
	flag.Parse()

	s := &state{staticGraph: &servicegraph.Static{Nodes: make(map[string]struct{})}}

	// don't allow directory listing
	jf := &justFilesFilesystem{http.Dir(*assetDir)}
	http.Handle("/", http.FileServer(jf))
	http.Handle("/graph", promgen.NewPromHandler(*promAddr, s.staticGraph, writeJSON))
	http.HandleFunc("/node", s.addNode)
	http.Handle("/dotgraph", promgen.NewPromHandler(*promAddr, s.staticGraph, dot.GenerateRaw))
	http.Handle("/dotviz", promgen.NewPromHandler(*promAddr, s.staticGraph, dot.GenerateHTML))
	http.Handle("/d3graph", promgen.NewPromHandler(*promAddr, s.staticGraph, servicegraph.GenerateD3JSON))
	http.Handle("/vizgraph", promgen.NewPromHandler(*promAddr, s.staticGraph, servicegraph.GenerateVizJSON))
	http.HandleFunc("/vizSocket", vizSocket)

	log.Printf("Starting servicegraph service at %s", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, nil))
}
