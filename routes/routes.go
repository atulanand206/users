package routes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/atulanand206/go-mongo"
	"github.com/atulanand206/users/objects"
	"go.mongodb.org/mongo-driver/bson"
	mg "go.mongodb.org/mongo-driver/mongo"
)

const (
	contentTypeKey             = "content-type"
	cors                       = "Access-Control-Allow-Origin"
	contentTypeApplicationJson = "application/json"
)

var database string
var collection string

func Routes() *http.ServeMux {
	database = os.Getenv("DATABASE")
	collection = os.Getenv("MONGO_COLLECTION")

	router := http.NewServeMux()
	router.HandleFunc("/user", http.HandlerFunc(newUserHandler))
	router.HandleFunc("/users", http.HandlerFunc(getUsersHandler))
	router.HandleFunc("/users/username/", http.HandlerFunc(getUserByUsernameHandler))
	router.HandleFunc("/authorize", http.HandlerFunc(authorizeHandler))
	return router
}

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ob objects.UserRequest
	err := decoder.Decode(&ob)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusBadRequest)
		return
	}
	ob.EncryptedPassword = hash(ob.Password)
	document, err := document(&ob)
	if err != nil {
		http.Error(w, "Can't create DB request", http.StatusInternalServerError)
		return
	}
	response, err := mongo.Write(database, collection, *document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(cors, "*")
	w.Header().Set(contentTypeKey, contentTypeApplicationJson)
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
	var response []objects.User
	cursor, err := mongo.Find(database, collection, bson.M{})
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
	w.Header().Set(cors, "*")
	w.Header().Set(contentTypeKey, contentTypeApplicationJson)
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
	w.Header().Set(cors, "*")
	w.Header().Set(contentTypeKey, contentTypeApplicationJson)
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
	w.Header().Set(cors, "*")
	w.Header().Set(contentTypeKey, contentTypeApplicationJson)
}

func hash(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}
