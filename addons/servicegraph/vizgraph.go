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

// Package servicegraph defines the core model for the servicegraph service.
package servicegraph

import (
	"encoding/json"
	"io"
	"log"
	"strconv"
	"time"
)

type (
	vizMetrics struct {
		Normal float64  `json:"normal"`
		Danger float64  `json:"danger"`
	}

	vizConnection struct {
		Source string      `json:"source"`
		Target string      `json:"target"`
		Class string       `json:"class,omitempty"`
		Metrics vizMetrics `json:"metrics,omitempty"`
	}

	vizNode struct {
		Name string                 `json:"name"`
		Renderer string             `json:"renderer,omitempty"`
		Class string                `json:"class,omitempty"`
		Nodes []vizNode             `json:"nodes,omitempty"`
		Connections []vizConnection `json:"connections,omitempty"`
		Updated int64               `json:"updated,omitempty"`
		MaxVolume float64           `json:"maxVolume,omitempty"`
	}
)

func GenerateVizJSON(w io.Writer, g *Dynamic) error {
	n := vizNode {
		Name: "edge",
		Renderer: "global",
		Nodes: make([]vizNode, 0, 2),
		Connections: make([]vizConnection, 0, 1),
	}
	n.Nodes = append(n.Nodes, vizNode{
		Name: "INTERNET",
		Renderer: "region",
		Class: "normal",
	})

	istNode := vizNode{
		Name: "k8s-ist-1",
		Renderer: "region",
		Class: "normal",
		Updated: time.Now().UnixNano() / 1000000,
		MaxVolume: 1000,
		Nodes: make([]vizNode, 0, len(g.Nodes)),
		Connections: make([]vizConnection, 0, len(g.Edges)),
	}
	for k := range g.Nodes {
		n := vizNode{
			Name: k,
		}
		istNode.Nodes = append(istNode.Nodes, n)
	}

	// log.Print(g.Edges)
	var overallIstioRps = 0.0
	for _, v := range g.Edges {
		// log.Print(v.Labels)

		var rps float64
		rps = 0.0
		rpsStr, ok := v.Labels["reqs/sec"]
		if ok {
			rpsParsed, err := strconv.ParseFloat(rpsStr, 64)
			if err == nil {
				rps = rpsParsed
				if v.Target == "istio-ingress.istio-system (unknown)" {
					overallIstioRps	= rps
				}
				log.Print(v.Source, v.Target, rps)
			}
		}

		l := vizConnection {
			Source: v.Source,
			Target: v.Target,
			Metrics: vizMetrics {
				// Normal: 999.7,
				// Danger: 100.3,
				Normal: rps * 100,
				Danger: 0.0,
			},
		}
		istNode.Connections = append(istNode.Connections, l)
	}

	n.Nodes = append(n.Nodes, istNode)
	n.Connections = append(n.Connections, vizConnection {
		Source: "INTERNET",
		Target: "k8s-ist-1",
		Metrics: vizMetrics {
			Normal: overallIstioRps * 100, // 26037.626,
			Danger: 0.0,
		},
	})

	return json.NewEncoder(w).Encode(n)
}
