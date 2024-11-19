package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PROXY_PORT")
	if port == "" {
		fmt.Println("PROXY_PORT environment variable is not set")
		os.Exit(1)
	}
	address := fmt.Sprintf(":%s", port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Fel vid skapande av listener:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Proxy servern lyssnar på %s\n", address)

	// Använd dynamisk serveradress och port från miljövariabler
	targetAddr := os.Getenv("TARGET_ADDRESS")
	if targetAddr == "" {
		fmt.Println("TARGET_ADDRESS environment variable is not set")
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Fel vid anslutning:", err)
			continue
		}
		go handleRequest(conn, targetAddr)
	}
}

func handleRequest(conn net.Conn, targetAddr string) {
	defer conn.Close()

	// Använd net/http's ReadRequest för att parsa HTTP-förfrågan från klienten
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Printf("Error reading client request: %v\n", err)
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}

	// Hantera endast GET-förfrågningar
	if request.Method != http.MethodGet {
		fmt.Printf("Received unsupported method: %s\n", request.Method)
		sendErrorResponse(conn, 501, "Not Implemented")
		return
	}

	// Skapa en ny förfrågan för att vidarebefordra till målservern
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Printf("Error connecting to target server: %v\n", err)
		sendErrorResponse(conn, 502, "Bad Gateway")
		return
	}
	defer targetConn.Close()

	// Skriv förfrågan till målservern
	err = request.Write(targetConn)
	if err != nil {
		fmt.Printf("Error forwarding request: %v\n", err)
		sendErrorResponse(conn, 502, "Bad Gateway")
		return
	}

	// Läs svar från målservern
	resp, err := http.ReadResponse(bufio.NewReader(targetConn), request)
	if err != nil {
		fmt.Printf("Error reading response from target server: %v\n", err)
		sendErrorResponse(conn, 502, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// Vidarebefordra svaret tillbaka till klienten
	err = resp.Write(conn)
	if err != nil {
		fmt.Printf("Error writing response to client %v\n", err)
	}
}

// Skickar ett felsvar med en given statuskod och statusmeddelande
func sendErrorResponse(conn net.Conn, statusCode int, statusText string) {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		statusCode, statusText, len(statusText), statusText)
	conn.Write([]byte(response))
}
