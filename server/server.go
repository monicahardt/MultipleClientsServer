package main

import (
	proto "Multipleclientsserver/grpc"
	"context"
	"flag"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedChittyChatServer
	port          int
	LamportClock   int64
	clientsStreams 	map[string]*proto.ChittyChat_JoinServer
}

var port = flag.Int("port", 0, "server port number") // create the port that recieves the port that the client wants to access to

func main() {
	flag.Parse()

	server := &Server{
		port: *port,
		LamportClock: 0,
		clientsStreams: make(map[string]*proto.ChittyChat_JoinServer),
	}

	go startServer(server)

	for {
	}
}


func startServer(server *Server) {
	grpcServer := grpc.NewServer()                                           // create a new grpc server
	listen, err := net.Listen("tcp", "localhost:"+strconv.Itoa(server.port)) // creates the listener

	if err != nil {
		log.Fatalln("Could not start listener")
	}

	log.Printf("Server started at port %v", server.port)

	proto.RegisterChittyChatServer(grpcServer, server)
	serverError := grpcServer.Serve(listen)

	if serverError != nil {
		log.Printf("Could not register server")
	}
}

//when a client joins the chat it recieves a stream so that it can send messages until that stream closes
func (s *Server) Join(request *proto.JoinRequest, stream proto.ChittyChat_JoinServer) error {
	//adding the stream to the client to the servers array of streams
	s.clientsStreams[request.User] = &stream

	joinMessage:= &proto.Message{
		User: request.User,
		Message: "User " + request.User + " has joined ChittyChat",
		LamportClock: s.LamportClock,
	}

	//Sends the join message to all clients connection via a stream
	for _, stream := range s.clientsStreams {
		(*stream).Send(joinMessage)
	}

	// waits for the stream to be closed -- happens when the client stops
	// then removes the stream from the streams map
	// and sends a message to the other clients
	<-stream.Context().Done()
	delete(s.clientsStreams, request.User)
	log.Println(request.User, "left the chat")

	leaveMessage := &proto.Message{
		User:         request.User,
		Message:      request.User + " has left the chat",
		LamportClock: s.LamportClock,
	}

	//Sends the leave message to all clients connection via a stream
	for _, stream := range s.clientsStreams {
		(*stream).Send(leaveMessage)
	}
	return nil
}	


func (s *Server) Publish(ctx context.Context, message *proto.Message) (*proto.Empty, error){
	if message.LamportClock < s.LamportClock {
		message.LamportClock = s.LamportClock + 1
	}

	// update the Lamport time of the server
	s.LamportClock = message.LamportClock

	//Sends the leave message to all clients connection via a stream
	for _, stream := range s.clientsStreams {
		(*stream).Send(message)
	}

	return &proto.Empty{}, nil
}
