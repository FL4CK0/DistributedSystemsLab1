Basic Part:
Hur vi kör koden:

SERVER_PORT=xxxx docker-compose up --build server

Testa med POSTMAN i ordning,

Bonus Part:
Hur vi kör koden: 

SERVER_PORT=8080 PROXY_PORT=8081 docker-compose up --build

Testa med:

curl -X GET http://localhost:8080/index.html -x http://localhost:8081

curl -X GET http://localhost:8080/Test123.txt -x http://localhost:8081

PUT Not implemented : curl -X PUT http://localhost:8080/index.html -x http://localhost:8081

POST Not implemented : curl -X POST http://localhost:8080/upload -d "Test123" -x http://localhost:8081 -H "Content-Type: text/plain"




För att köra denna kod genom docker på en annan dator:
docker login
docker pull uffep/distruberadesystem-server:latest
docker pull uffep/distruberadesystem-proxy:latest

Skapa en docker-compose.yml fil med detta:


services:
  proxy:
    image: uffep/distruberadesystem-proxy:latest
    ports:
      - "${PROXY_PORT:-8081}:8081"
    environment:
      - PROXY_PORT=${PROXY_PORT:-8081}
    depends_on:
      - server
    networks:
      - proxy_network

  server:
    image: uffep/distruberadesystem-server:latest
    ports:
      - "${SERVER_PORT:-8080}:8080"
    environment:
      - SERVER_PORT=${SERVER_PORT:-8080}
    volumes:
      - ./uploads:/app/uploads
    networks:
      - proxy_network

networks:
  proxy_network:
    driver: bridge



Sen kör detta kommand: 

SERVER_PORT=8080 PROXY_PORT=8081 docker-compose up
