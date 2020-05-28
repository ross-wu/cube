//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	serverAddr = flag.String("server", "", "Cube-solver server address and port.")
)

func main() {
	flag.Parse()

	if *serverAddr == "" {
		fmt.Println("ERROR: --server must be set.")
		os.Exit(1)
	}

	req := fmt.Sprintf("http://%s/cube?U=yyoyygbwo&L=ggwooboob&F=rrwybwyoo&R=brgbrgyrg&B=wrrwgywoy&D=rbbgwbgwr", *serverAddr)
	log.Printf("GET %s", req)
	resp, err := http.Get(req)
	if err != nil {
		log.Printf("ERROR: GET error: %v", err)
		os.Exit(255)
	}

	fmt.Printf("Response:\nStatus: %d: %s\n\n", resp.StatusCode, resp.Status)
	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("ERROR: read body error: %v", err)
	}
}
