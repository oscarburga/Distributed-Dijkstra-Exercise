/*
Run with argument "localhost:port". The program will listen on that port.
	Example: go run dijkstra.go localhost:8005

To add an edge from this node to another one, input through stdin:
	"edge host:port cost"
	- Example: edge localhost:8002 12
		-> This adds a directed edge from this to localhost:8002 with cost 12.

To run the Dijkstra algorithm with this node as the source, simply input
	"Dijkstra" (through stdin).

After running Dijkstra, the distance values for the shortest path are printed
	on the very last node that was processed.

Note:
	This program only works for 1 run of the algorithm. Any operation after
	running the algorithm is not being thoughtfully handled and anything could
	happen.
	This code was written to be ran once and shut down everything after. Should
	you wish to run it again, then restart all the nodes and re-create the
	edges.
*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

const CmdCreateEdge = "CreateEdge"
const CmdProcessNode = "ProcessNode"

// Probably should use slice of byte slices instead of strings.
type Frame struct {
	Cmd    string   `json:"cmd"`
	Sender string   `json:"sender"`
	Data   []string `json:"data"`
}

type Edge struct {
	remote string
	cost   int
}

var chEdges chan []Edge        // "Adjacency list" for this node
var chCost chan map[string]int // Best cost to each node from the source
var chVis chan map[string]bool // Visited nodes so far
var host string                // This node's address

// Didn't get to implement parents due to the tight time limit.
// var chParents chan map[string]string

// Confirm received edge (doesn't actually do anything)
func ReceiveCreateEdge(frame *Frame) {
	log.Printf("%s: Created edge from %s\n", host, frame.Sender)
}

// Create edge from this to another node (ping it to ensure connectivity)
func SendCreateEdge(edge Edge) {
	log.Printf("%s: Sending create edge to %s with cost %d",
		host, edge.remote, edge.cost)

	msg := Frame{CmdCreateEdge, host, []string{strconv.Itoa(edge.cost)}}
	if send(edge.remote, msg, nil) {
		edges := <-chEdges
		edges = append(edges, edge)
		log.Printf("%s: Created Edge to %s with cost %d\n",
			host, edge.remote, edge.cost)
		chEdges <- edges
	}
}

// Receive this node's turn to be processed in the execution of Dijkstra.
func HandleProcessThisNode(frame *Frame) {
	// If properly implemented, then this only gets called once per Dijkstra.
	costData := frame.Data[0]
	visData := frame.Data[1]
	// parentData := msg.Data[2]
	order, _ := strconv.Atoi(frame.Data[3])
	log.Printf("%s: Handling process this node ~ its the %d'th node \n",
		host, order)

	costMap := make(map[string]int)
	json.Unmarshal([]byte(costData), &costMap)
	log.Printf("%s: costs= %v\n", host, costMap)
	chCost <- costMap

	visMap := make(map[string]bool)
	json.Unmarshal([]byte(visData), &visMap)
	log.Printf("%s: vis= %v\n", host, visMap)
	chVis <- visMap

	ProcessThisNode(order)
}

// Its this node's turn to be processed. Update distances and send next turn.
func ProcessThisNode(order int) {
	log.Printf("%s: Processing this node - order %d\n", host, order)
	edges := <-chEdges
	vis := <-chVis
	costs := <-chCost
	log.Printf("%s: ~ Doing work ~ \n", host)
	vis[host] = true
	myCost := costs[host]
	// Check adjacent nodes and update costs.
	for _, edge := range edges {

		newDist := myCost + edge.cost
		_, found := vis[edge.remote]
		if !found {
			vis[edge.remote] = false
			costs[edge.remote] = newDist
		}

		if vis[edge.remote] {
			continue
		}

		if newDist < costs[edge.remote] {
			costs[edge.remote] = newDist
		}
	}
	log.Printf("%s: ~ Done updating ~ \n", host)

	// Choose the next best node.
	bestNode := ""
	bestCost := 100000000 // 10^8, treating this as infinity
	for remote, cost := range costs {
		if !vis[remote] && cost < bestCost {
			bestCost = cost
			bestNode = remote
		}
	}
	log.Printf("%s: ~ Best Node: %s with cost %d ~ \n", host, bestNode, bestCost)

	if bestNode != "" {
		// Notify the next best node's turn.
		// Have to send all the network data known so far.
		// Possible small optimization: only send costs for non-visited nodes.
		costData, _ := json.Marshal(costs)
		visData, _ := json.Marshal(vis)
		costStr := string(costData)
		visStr := string(visData)
		orderStr := strconv.Itoa(order + 1)

		msg := Frame{}
		msg.Cmd = CmdProcessNode
		msg.Sender = host
		msg.Data = []string{costStr, visStr, "", orderStr}
		log.Printf("%s: Sending process signal to %s\n", host, bestNode)
		send(bestNode, msg, nil)
		log.Printf("%s: Sent process signal to %s\n", host, bestNode)
	} else {
		log.Printf("No next node! Dijsktra done\n")
		for node, cost := range costs {
			log.Printf("%s: Cost of %d to get here.", node, cost)
		}
	}
	chEdges <- edges
	chCost <- costs
	chVis <- vis
	log.Printf("%s: Done processing this node\n", host)
}

// Start Dijkstra from this node
func StartDijkstra() {
	log.Printf("%s: Starting dijkstra\n", host)
	chVis <- make(map[string]bool)
	cost := make(map[string]int)
	cost[host] = 0
	chCost <- cost

	ProcessThisNode(1)
}

// Dispatcher for receiving connections.
func ProcessConnection(cn net.Conn) {
	defer cn.Close()
	dec := json.NewDecoder(cn)
	frame := &Frame{}
	dec.Decode(frame)
	log.Printf("Processing connection with cmd %s from %s", frame.Cmd, frame.Sender)
	switch frame.Cmd {
	case CmdCreateEdge:
		ReceiveCreateEdge(frame)
	case CmdProcessNode:
		HandleProcessThisNode(frame)
	}
}

// Server/Listener.
func netListener() {
	if ln, err := net.Listen("tcp", host); err == nil {
		defer ln.Close()
		log.Printf("Listening on %s\n", host)
		for {
			if cn, err := ln.Accept(); err == nil {
				go ProcessConnection(cn)
			} else {
				log.Printf("%s: cant accept connection.\n", host)
			}
		}
	} else {
		log.Printf("Can't listen on %s\n", host)
	}
}

// Initialize stuff and read user input from stdin to add edges and run the alg.
func main() {
	if len(os.Args) != 2 {
		panic("Need exactly 1 argument: 'localhost:port'")
	}
	host = os.Args[1]
	chEdges = make(chan []Edge, 1)
	chEdges <- []Edge{}
	chVis = make(chan map[string]bool, 1)
	chCost = make(chan map[string]int, 1)
	go netListener()
	fmt.Printf("\n------------\n")
	fmt.Printf("write 'edge host:port cost' to add an edge\n")
	fmt.Printf("write 'Dijkstra' to run the alg. with this node as the source")
	fmt.Printf("\n------------\n\n")
	for {
		var op, to string
		var cost int
		fmt.Scanf("%s", &op)
		if op == "edge" {
			fmt.Scanf("%s %d", &to, &cost)
			newEdge := Edge{to, cost}
			SendCreateEdge(newEdge)
		} else if op == "Dijkstra" {
			log.Printf("%s: Starting dijsktra...\n", host)
			StartDijkstra()
		}
	}
}

// Simple generic function to send data.
func send(remote string, frame Frame, callback func(net.Conn)) bool {
	if cn, err := net.Dial("tcp", remote); err == nil {
		defer cn.Close()
		enc := json.NewEncoder(cn)
		enc.Encode(frame)
		if callback != nil {
			callback(cn)
		}
		return true
	}
	log.Printf("%s: failed to connect to %s\n", host, remote)
	return false
}
