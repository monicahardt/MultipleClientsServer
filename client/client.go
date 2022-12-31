package main

import (
	proto "Multipleclientsserver/grpc"
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	id int
	name string
	lamportClock int64      //the lamport clock
}

var (
	clientPort = flag.Int("cPort", 0, "client port number")
	serverPort = flag.Int("sPort", 0, "server port number (should match the port used for the server)")
	clientName = flag.String("name", "", "client name (can be any name)")
	server proto.ChittyChatClient
)

func main() {
	//setting the log file
	f, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Parse the flags to get the port for the client
	flag.Parse()
	
	//Create a client
	client := &Client{
		id: *clientPort,
		lamportClock: int64(0),
		name: *clientName,
	}

	// Wait for the client (user) to ask for the time
	server, err = connectToServer()
	go client.joinChat()
	go client.chat()

	for {
	}
}

func connectToServer() (proto.ChittyChatClient, error) {
	// Dial the server at the specified port.
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the server at port %d\n", *serverPort)
		fmt.Printf("Connected to the server at port %d\n", *serverPort)
	}

	return proto.NewChittyChatClient(conn), nil
}

func (c *Client) joinChat(){
	joinRequest := &proto.JoinRequest{
		User:         *clientName,
		LamportClock: 0,
	}
	//when we join the chat we recieve a stream back from the server
	stream,_ := server.Join(context.Background(),joinRequest)
	
	for {
		select {
		case <-stream.Context().Done():
			fmt.Println("Connection to server closed")
			return // stream is done
		default:
		}

		incomingMessage, err := stream.Recv()

		if err == io.EOF {
			fmt.Println("Server is done sending messages")
			return
		}
		if err != nil {
			log.Fatalf("Failed to receive message from channel. \nErr: %v", err)
		}

		if incomingMessage.LamportClock > c.lamportClock {
			c.lamportClock = incomingMessage.LamportClock + 1
		} else {
			c.lamportClock++
		}

		log.Printf("%s got message from %s: %s", *clientName, incomingMessage.User, incomingMessage.Message)
		fmt.Printf("\rLamport: %v | %v: %v \n", c.lamportClock, incomingMessage.User, incomingMessage.Message)
		fmt.Print("-> ")
	}
}

	func (c *Client) chat(){
	scanner := bufio.NewReader(os.Stdin)
	//Infinite loop to listen for clients input.
	fmt.Print("-> ")
	for {
		//Read user input to the next newline
		input, err := scanner.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim whitespace

		// we are sending a message so we increment the lamport clock
		c.lamportClock++

		// send the message in the chat
		response, err := server.Publish(context.Background(), &proto.Message{
			User:         *clientName,
			Message:      input,
			LamportClock: c.lamportClock,
		})

		if err != nil || response == nil {
			log.Printf("Client %s: something went wrong with the server :(", *clientName)
			continue
		}
	}
}