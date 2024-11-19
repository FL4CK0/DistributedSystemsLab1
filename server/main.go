package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

var sem = make(chan struct{}, 10) // Begränsar till max 10 samtidiga förfrågningar

// Huvudfunktionen för att starta servern
func main() {

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		fmt.Println("SERVER_PORT environment variable is not set")
		os.Exit(1)
	}
	address := fmt.Sprintf(":%s", port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Fel vid skapande av listener:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Lyssnar på localhost %s\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Fel vid anslutning:", err)
			continue
		}

		sem <- struct{}{} // Blockerar om max antal gorutiner nås
		go func() {
			handleConnection(conn)
			<-sem // Släpper semaforen efter hantering
		}()
	}
}

// Hanterar varje anslutning
func handleConnection(conn net.Conn) {
	defer conn.Close()

	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}

	switch request.Method {
	case "GET":
		handleGet(conn, request)
	case "POST":
		handlePost(conn, request)
	default:
		sendErrorResponse(conn, 501, "Not Implemented")
	}
}

// Hanterar GET-förfrågningar
func handleGet(conn net.Conn, request *http.Request) {
	path := request.URL.Path[1:] // Tar bort ledande "/"

	// Om path är tomt, sätt det till index.html
	if path == "" {
		path = "index.html"
	}

	// Kontrollera om förfrågan är för favicon.ico
	if path == "favicon.ico" {
		sendFavicon(conn)
		return
	}

	ext := filepath.Ext(path)
	if !isValidFileType(ext) {
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}

	// Skapa fullständig sökväg till filen i uploads-mappen
	filePath := filepath.Join("uploads", path)

	// Kontrollera om filen faktiskt finns
	data, err := os.ReadFile(filePath)
	if err != nil {
		sendErrorResponse(conn, 404, "Not Found")
		return
	}

	contentTypeHeader := contentType(ext)

	// Skicka svar med rätt Content-Type och filens data
	sendResponse(conn, 200, contentTypeHeader, data)
}

// Skickar favicon.ico om den finns
func sendFavicon(conn net.Conn) {
	faviconPath := filepath.Join("uploads", "favicon.ico") // Ändra sökvägen till "uploads/favico.ico"
	favicon, err := os.ReadFile(faviconPath)
	if err != nil {
		fmt.Println("Error reading favicon.ico:", err)
		sendErrorResponse(conn, 404, "Not Found")
		return
	}

	// Skicka favicon-svar med korrekt innehållstyp
	sendResponse(conn, 200, "image/x-icon", favicon)
}

// Hanterar POST-förfrågningar och hanterar filuppladdning
func handlePost(conn net.Conn, request *http.Request) {
	fmt.Println("Handling POST request")

	// Läs in formdata
	err := request.ParseMultipartForm(10 << 20) // Max 10MB
	if err != nil {
		fmt.Println("Failed to parse multipart form:", err)
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}

	fmt.Println("Multipart form parsed successfully")

	// Hämta filen från formdata
	file, fileHeader, err := request.FormFile("file")
	if err != nil {
		fmt.Println("Failed to get form file:", err)
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}
	defer file.Close()

	// Kontrollera filtypen
	ext := filepath.Ext(fileHeader.Filename)
	if !isValidFileType(ext) {
		fmt.Println("Invalid file type:", ext)
		sendErrorResponse(conn, 400, "Bad Request")
		return
	}

	// Använd det ursprungliga filnamnet från filhuvudet
	uploadedFileName := fileHeader.Filename

	fmt.Printf("Received file: %s\n", uploadedFileName)

	// Bestäm var filen ska sparas
	filePath := filepath.Join("uploads", uploadedFileName)

	// Skapa en fil för att spara den uppladdade filen
	outFile, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Failed to create file:", err)
		sendErrorResponse(conn, 500, "Internal Server Error")
		return
	}
	defer outFile.Close()

	// Kopiera innehållet från den uppladdade filen till den sparade filen
	_, err = io.Copy(outFile, file)
	if err != nil {
		fmt.Println("Failed to copy file:", err)
		sendErrorResponse(conn, 500, "Internal Server Error")
		return
	}

	// Skicka svar
	sendResponse(conn, 200, "text/plain", []byte("File uploaded successfully"))
	fmt.Println("File uploaded successfully")
}

// Validerar filtyp
func isValidFileType(ext string) bool {
	validExtensions := map[string]bool{
		".html": true,
		".txt":  true,
		".gif":  true,
		".jpeg": true,
		".jpg":  true,
		".css":  true,
	}
	return validExtensions[ext]
}

// Returnerar content type för en given filändelse
func contentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}

// Skickar en HTTP-svar med statuskod, content-type och data
func sendResponse(conn net.Conn, statusCode int, contentType string, data []byte) {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))
	headers := fmt.Sprintf("Content-Type: %s\r\nContent-Length: %d\r\n\r\n", contentType, len(data))
	conn.Write([]byte(statusLine + headers))
	conn.Write(data)
}

// Skickar ett felsvar med en given statuskod och statusmeddelande
func sendErrorResponse(conn net.Conn, statusCode int, statusText string) {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		statusCode, statusText, len(statusText), statusText)
	conn.Write([]byte(response))
}
