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
		Updated: time.Now().Unix(),
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

	vizceralScalingFactor := 10.0

	connectionsMap := make(map[string]*vizConnection)
	var overallIstioNormRPS = 0.0
	var overallIstioErrRPS = 0.0
	for _, v := range g.Edges {
		// log.Print(v.Source, v.Target, v.Labels)
		// log.Printf("%+v\n", v)

		var metrics vizMetrics
		rpsStr, ok := v.Labels["reqs/sec"]
		if ok {
			normRPSParsed, err := strconv.ParseFloat(rpsStr, 64)
			if err == nil {
				metrics.Normal = normRPSParsed * vizceralScalingFactor
				if v.Target == "istio-ingress.istio-system (unknown)" {
					overallIstioNormRPS	= normRPSParsed
					log.Printf("Istio Normal RPS %f, scaled to %f", overallIstioNormRPS, overallIstioNormRPS * vizceralScalingFactor)
				}
			}
		}
		epsStr, ok := v.Labels["errs/sec"]
		if ok {
			errRPSParsed, err := strconv.ParseFloat(epsStr, 64)
			if err == nil {
				metrics.Danger = errRPSParsed * vizceralScalingFactor
				if v.Target == "istio-ingress.istio-system (unknown)" {
					overallIstioErrRPS	= errRPSParsed
					log.Printf("Istio Error RPS %f, scaled to %f", overallIstioErrRPS, overallIstioErrRPS * vizceralScalingFactor)
				}
			}
		}

		l := vizConnection {
			Source: v.Source,
			Target: v.Target,
			Metrics: metrics,
		}

		// Merge the new vizConnection into map of existing connections
		connKey := v.Source + v.Target
		existing, found := connectionsMap[connKey]
		if found {
			// log.Printf("Found %s in map", connKey)
			if (l.Metrics.Normal > existing.Metrics.Normal) {
				existing.Metrics.Normal = l.Metrics.Normal
			}
			if (l.Metrics.Danger > existing.Metrics.Danger) {
				existing.Metrics.Danger = l.Metrics.Danger
			}
			connectionsMap[connKey].Metrics = existing.Metrics
		} else {
			// log.Printf("Added %s to map", connKey)
			connectionsMap[connKey] = &l
		}
	}

	// Take values from resulting map and add to array of Connections
	for _, value := range connectionsMap {
		istNode.Connections = append(istNode.Connections, *value)
	}

	n.Nodes = append(n.Nodes, istNode)
	n.Connections = append(n.Connections, vizConnection {
		Source: "INTERNET",
		Target: "k8s-ist-1",
		Metrics: vizMetrics {
			Normal: overallIstioNormRPS * vizceralScalingFactor,
			Danger: overallIstioErrRPS * vizceralScalingFactor,
		},
	})

	return json.NewEncoder(w).Encode(n)
}
