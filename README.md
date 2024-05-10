# sjsu-pub-sub
Distributed publish-subscribe system

by Riddhik Tilawat, Fahad Siddiqui, and Pawan Sarma

## Introduction

Distributed system code for servers, a server gateway, and clients. System is capable of efficient P2P gossip and demonstrates high availability due to leader election and data replication algorithms occuring in backend. 

## Our testing

We deployed our servers on Google Cloud Platform (GCP) to properly simulate a real distributed environment and each server node ran on a different server. We did this by creating a standard VM and spinning up multiple VMs, each hosting a server. We also deployed our gateway server on GCP on a separate VM. We initially thought to run our clients locally, but due to connection issues over the firewall, our clients were able to connect to the servers but not the other way around. As a workaround, we deployed VMs and spun up clients remotely. We were then able to test our system and the 3 distributed algorithms we implemented.

## Deployment steps

1. Create VMs in GCP:
    - Log in to GCP console and navigate to the Compute Engine
    - Create a new VM instance and configure it with the required specifications (e.g., OS, disk size, memory).
    - SSH into VM once running

2. Run gateway in VM:
    - `go run gateway.go` to start up gateway. It will register servers joining as they are spun up.
        -  Gateway runs on TCP port 8087 and HTTP port 8080

3. Run one (or more) servers in respective VMs:
    - `go run server.go` to start up a server. Run the same command in other VMs to start multiple servers.
        - Servers run on TCP ports 8081 and 8082 and HTTP port 8080
    - **Spin up a MongoDB instance on each VM where a server is running and ensure its running on `mongodb://localhost:27017`**

4. Run one (or more) clients in respective VMs:
    - `go run client.go` to start up a client. Run the same command in other VMs to start multiple clients.
        - Clients run on a randomly generated port from 5000-9999 for TCP

## Testing functionalities

1. Basic client functionalities:
    - Spin up gateway, server(s) with running MongoDB instance(s), and client(s) in that order
    - See all groups: Enter 1
    - Join a group: Enter 2 and provide group name
    - Write a post to a group: Enter 3, provide the group to write the post to, and write the post

2. Gossip:
    - Spin up gateway, server(s) with running MongoDB instance(s), and client(s) in that order. For gossip, true functionality is 
    observable with many clients (10+)
    - Ensure all clients have joined some group G
    - Write post from some client to group G. Gossip will spread to all clients
    - **Testing failure tolerance:** Repeat above steps and bring down any number of clients after the gossip starts. The gossip will still spread to all active clients

3. Leader election:
    - Spin up gateway
    - Spin up one server with running MongoDB instance. Observe that is has been elected as leader
    - Spin up more server(s) and see that server with lowest hostname is elected as leader
    - Bring down one server (ctrl + C) and observe that leader election is triggered

4. Replication:
    - Spin up gateway, server(s) with running MongoDB instance(s), and client(s) in that order
    - Write post from some client to group G. This update will be written to all active servers
    - Bring down leader node using ctrl + C and observe that leader election is triggered. A new leader will be elected
    - Get all groups from some client. Observe that despite having a new leader running with a different DB, the post written above is reflected in this DB as well.


## Division of work

We all met up frequently held working sessions together to design and implement (code) our system. Fahad additionally took 
the responsibility of getting familiar Google Cloud Platform and deploying our servers remotely.