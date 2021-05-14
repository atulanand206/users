package routes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/atulanand206/go-mongo"
	net "github.com/atulanand206/go-network"
	"github.com/atulanand206/users/objects"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mg "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var database string
var collection string

func Routes() *http.ServeMux {
	database = os.Getenv("DATABASE")
	collection = os.Getenv("MONGO_COLLECTION")

	authChain := net.MiddlewareChain{
		net.CorsAllInterceptor(),
		net.ApplicationJsonInterceptor(),
		net.AuthenticationInterceptor(),
	}

	chain := net.MiddlewareChain{
		net.CorsAllInterceptor(),
		net.ApplicationJsonInterceptor(),
	}

	router := http.NewServeMux()
	router.HandleFunc("/user", chain.Handler(newUserHandler))
	router.HandleFunc("/users", authChain.Handler(getUsersHandler))
	router.HandleFunc("/users/username/", authChain.Handler(getUserByUsernameHandler))
	router.HandleFunc("/user/username/", authChain.Handler(updateUserHandler))
	router.HandleFunc("/authorize", chain.Handler(authorizeHandler))
	return router
}

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var userRequest objects.UserRequest
	err := decoder.Decode(&userRequest)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusBadRequest)
		return
	}
	userRequest.EncryptedPassword = hash(userRequest.Password)
	document, err := document(&userRequest)
	if err != nil {
		http.Error(w, "Can't create DB request", http.StatusInternalServerError)
		return
	}
	response, err := mongo.Write(database, collection, *document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(response)
}

func document(v interface{}) (doc *bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		log.Println(err)
	}
	err = bson.Unmarshal(data, &doc)
	return
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var usernames []string
	err := decoder.Decode(&usernames)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusInternalServerError)
		return
	}
	var response []objects.User
	opts := options.Find()
	opts.SetSort(bson.D{primitive.E{Key: "rating", Value: -1}})
	cursor, err := mongo.Find(database, collection,
		bson.M{"username": bson.M{"$in": usernames}}, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for cursor.Next(context.Background()) {
		var user objects.User
		err := cursor.Decode(&user)
		if err != nil {
			http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		}
		response = append(response, user)
	}
	json.NewEncoder(w).Encode(response)
}

func getUserByUsernameHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/users/username/")
	response := mongo.FindOne(database, collection, bson.M{"username": username})
	err := response.Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := decodeUser(response)
	if err != nil {
		http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(user)
}

func decodeUser(document *mg.SingleResult) (v objects.User, err error) {
	var user objects.User
	if err = document.Decode(&user); err != nil {
		log.Println(err)
		return user, err
	}
	return user, err
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	userId := strings.TrimPrefix(r.URL.Path, "/user/username/")
	uId, _ := primitive.ObjectIDFromHex(userId)
	fmt.Println(userId)
	decoder := json.NewDecoder(r.Body)
	var user objects.UpdateUser
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusBadRequest)
		return
	}
	document, err := document(&user)
	if err != nil {
		http.Error(w, "Can't create DB request", http.StatusInternalServerError)
		return
	}
	fmt.Println(*document)
	response, err := mongo.Update(database, collection,
		bson.M{"_id": uId}, bson.D{primitive.E{Key: "$set", Value: *document}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(response)
	json.NewEncoder(w).Encode(response)
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ob objects.AuthorizeRequest
	err := decoder.Decode(&ob)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusInternalServerError)
		return
	}
	response := mongo.FindOne(database, collection,
		bson.M{"username": ob.Username, "password": hash(ob.Password)})
	err = response.Err()
	if err != nil {
		http.Error(w, "Invalid login", http.StatusInternalServerError)
		return
	}
	token, err := net.CreateToken(ob.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(token)
}

func hash(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}
