# Distributed Dijkstra SSSP exercise

This is a 2-hour implementation of a "Distributed Dijkstra" shortest path 
exercise. It was part of an university exam and I thought it was kinda cool.

The given constraint for the exercise is that this node can only access its 
own data, and the data it receives from other nodes. 

This solution is built upon the idea that every running instance of this 
program is a node in the graph, and it will listen on some port. The nodes then 
have to send messages between themselves to calculate the shortest paths from 
the source node to all others. 

The idea is the same as classical Dijkstra algorithm, but instead of popping 
from a priority queue and processing the next node, a message is sent to the 
next node with all the current information about the visited nodes and costs, 
and then that node is processed.