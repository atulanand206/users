package main

import (
	"net/http"
	"os"

	"github.com/atulanand206/go-mongo"
	"github.com/atulanand206/users/routes"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// Register the MongoDB cloud atlas database.
	mongoClientId := os.Getenv("MONGO_CLIENT_ID")
	mongo.ConfigureMongoClient(mongoClientId)

	// Register the endpoints exposed from the service.
	routes := routes.Routes()
	handler := http.HandlerFunc(routes.ServeHTTP)
	port := os.Getenv("PORT")
	http.ListenAndServe(":"+port, handler)
}
