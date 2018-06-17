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
	"errors"
	"io"
	"log"
	"os"
)

type (
	// vizzzGraph is a graph representation for JSON serialization to be
	// consumed easily by the viz.js library.
	vizzzGraph struct {
		Nodes []vizzzNode `json:"nodes"`
		Links []vizzzLink `json:"links"`
	}

	vizzzNode struct {
		Name string `json:"name"`
	}

	vizzzLink struct {
		Source int        `json:"source"`
		Target int        `json:"target"`
		Labels Attributes `json:"labels"`
	}
)

func indexOfV(nodes []vizzzNode, name string) (int, error) {
	for i, v := range nodes {
		if v.Name == name {
			return i, nil
		}
	}
	return 0, errors.New("invalid graph")
}

// GenerateVizJSON converts the standard Dynamic graph to vizzzGraph, then
// serializes to JSON.
func GenerateVizJSON(w io.Writer, g *Dynamic) error {
	log.Print(g.Nodes)
	log.Print(g.Edges)
	graph := vizzzGraph{
		Nodes: make([]vizzzNode, 0, len(g.Nodes)),
		Links: make([]vizzzLink, 0, len(g.Edges)),
	}
	for k := range g.Nodes {
		n := vizzzNode{
			Name: k,
		}
		graph.Nodes = append(graph.Nodes, n)
	}
	for _, v := range g.Edges {
		s, err := indexOfV(graph.Nodes, v.Source)
		if err != nil {
			return err
		}
		t, err := indexOfV(graph.Nodes, v.Target)
		if err != nil {
			return err
		}
		l := vizzzLink{
			Source: s,
			Target: t,
			Labels: v.Labels,
		}
		graph.Links = append(graph.Links, l)
	}
	// log.Print(graph)
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(&graph)

	return json.NewEncoder(w).Encode(graph)
}

func GenerateVizJSON1(w io.Writer, g *Dynamic) error {
	arr := []byte("XYZ")
	_, err := w.Write(arr)
	return err
}

func GenerateVizJSON2(w io.Writer, g *Dynamic) error {
	reader, err := os.Open("/apps/go/src/istio.io/istio/addons/servicegraph/sample_data_simple.json")
	if (err != nil) {
		log.Fatal(err)
		return err
	}

	_, err = io.Copy(w, reader)
	if (err != nil) {
		log.Fatal(err)
		return err
	}

	return err
}