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
	"html/template"
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

func echo(w http.ResponseWriter, r *http.Request) {

	log.Print("In echo")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func vizSocket(w http.ResponseWriter, r *http.Request) {

	log.Print("In vizSocket")

	tickChan := time.NewTicker(time.Second * 2).C

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	doneChan := make(chan bool)
	defer func() {
		log.Print("in defer", err)
		c.Close()
		doneChan <- true
	}()
	for {
		select {
		case <- tickChan:
			log.Print("Ticker ticked");
			err = c.WriteMessage(websocket.TextMessage, []byte("NGK.."))
			if err != nil {
				log.Println("write:", err)
				break
			}
		case <- doneChan:
			log.Print("Done");
			return
		}
		log.Print("After select")
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
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
	http.Handle("/vizgraph1", promgen.NewPromHandler(*promAddr, s.staticGraph, servicegraph.GenerateVizJSON1))
	http.Handle("/vizgraph2", promgen.NewPromHandler(*promAddr, s.staticGraph, servicegraph.GenerateVizJSON2))
	http.Handle("/vizgraph3", promgen.NewPromHandler(*promAddr, s.staticGraph, servicegraph.GenerateVizJSON3))
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/h", home)
	http.HandleFunc("/vizSocket", vizSocket)

	log.Printf("Starting servicegraph service at %s", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server,
"Send" to send a message to the server and "Close" to close the connection.
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
