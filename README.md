# sjsu-pub-sub
Distributed publish-subscribe system

by Riddhik Tilawat, Fahad Siddiqui, and Pawan Sarma

## Our testing

We deployed our servers on Google Cloud Platform (GCP), so our current client, server, and gateway code have IPs hardcoded. To test 

## Testing with our deployed servers

1. Setting up:
- `go run mockserver.go` to start server locally
- `go run client.go` to start client. Use same command to spin up multiple clients
    - Enter username
    - Enter 1 to see all groups, enter 2 to join a group, enter 3 to write a post

2. Testing Gossip:
- Spin up one server and multiple clients. Have active clients all join a group called "group1". Then write 
a post to this group from one client. You will see that all active clients 

