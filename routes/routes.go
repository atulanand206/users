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

	mg "github.com/atulanand206/go-mongo"
	net "github.com/atulanand206/go-network"
	"github.com/atulanand206/users/objects"
	"github.com/dgrijalva/jwt-go/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Instance variable to store the Database name.
var Database string

// Instance variable to store the DB Collection name.
var Collection string

func Routes() *http.ServeMux {
	Database = os.Getenv("DATABASE")
	Collection = os.Getenv("MONGO_COLLECTION")

	// Interceptor chain with authentication and json encoding.
	authenticatedChain := net.MiddlewareChain{
		net.ApplicationJsonInterceptor(),
		net.AuthenticationInterceptor(),
	}

	// Interceptor chain with only GET method.
	getChain := authenticatedChain.Add(net.CorsInterceptor(http.MethodGet))
	// Interceptor chain with only POST method.
	postChain := authenticatedChain.Add(net.CorsInterceptor(http.MethodPost))

	// Interceptor chain with json encoded POST method.
	chain := net.MiddlewareChain{
		net.ApplicationJsonInterceptor(),
		net.CorsInterceptor(http.MethodPost),
	}

	refreshTokenChain := net.MiddlewareChain{
		net.ApplicationJsonInterceptor(),
		net.CorsInterceptor(http.MethodPost),
		net.RefreshAuthenticationInterceptor(),
	}

	router := http.NewServeMux()
	// Endpoint for creating a new user.
	router.HandleFunc("/user", chain.Handler(HandlerNewUser))
	// Endpoint for getting users from a list of usernames.
	router.HandleFunc("/users", getChain.Handler(HandlerGetUsers))
	// Endpoint for getting user from a username.
	router.HandleFunc("/users/username/", getChain.Handler(HandlerGetUserByUsername))
	// Endpoint for updating a user.
	router.HandleFunc("/user/username/", postChain.Handler(HandlerUpdateUser))
	// Endpoint for authorizing a username and password.
	router.HandleFunc("/authorize", chain.Handler(HandlerAuthorize))
	// Endpoint for refreshing an access token.
	router.HandleFunc("/refresh", refreshTokenChain.Handler(HandlerRefreshToken))
	return router
}

// Handler for creating a new user.
func HandlerNewUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var userRequest objects.UserRequest
	// Decode the user information to create a new user.
	err := decoder.Decode(&userRequest)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusBadRequest)
		return
	}
	// Add the encrypted password to be used for verifying credentials.
	userRequest.EncryptedPassword = Hash(userRequest.Password)
	// Create the bson document for the mongo write request.
	document, err := mg.Document(&userRequest)
	if err != nil {
		http.Error(w, "Can't create DB request", http.StatusInternalServerError)
		return
	}
	// Write the new user to the mongo database.
	response, err := mg.Write(Database, Collection, *document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return the generated id as the json response.
	json.NewEncoder(w).Encode(response)
}

// Handler to get users for the leaderboard.
func HandlerGetUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var usernames []string
	// Decode the usernames from the request body.
	err := decoder.Decode(&usernames)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusInternalServerError)
		return
	}
	var response []objects.User
	// Create the sorting options for the db response.
	opts := options.Find()
	opts.SetSort(bson.D{primitive.E{Key: "rating", Value: -1}})
	// Find the users matching the given criteria.
	cursor, err := mg.Find(Database, Collection,
		bson.M{"username": bson.M{"$in": usernames}}, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Decode the cursor into an array of users.
	for cursor.Next(context.Background()) {
		var user objects.User
		err := cursor.Decode(&user)
		if err != nil {
			http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		}
		response = append(response, user)
	}
	// Returns the users as the json response.
	json.NewEncoder(w).Encode(response)
}

// Handler to get the user for the username.
func HandlerGetUserByUsername(w http.ResponseWriter, r *http.Request) {
	// Get the username from the url path variable.
	username := strings.TrimPrefix(r.URL.Path, "/users/username/")
	// Find the user from the database.
	response := mg.FindOne(Database, Collection, bson.M{"username": username})
	err := response.Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Decode user information received from the mongo response.
	user, err := DecodeUser(response)
	if err != nil {
		http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		return
	}
	// Returns the user object as a json encoded response.
	json.NewEncoder(w).Encode(user)
}

// Decodes a mongo db single result into an user object.
func DecodeUser(document *mongo.SingleResult) (v objects.User, err error) {
	var user objects.User
	if err = document.Decode(&user); err != nil {
		log.Println(err)
		return user, err
	}
	return user, err
}

// Handler for updating user into the mongo database.
func HandlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get the userId from the url path variable.
	userId := strings.TrimPrefix(r.URL.Path, "/user/username/")
	// Generate the object id from the hex userId.
	uId, _ := primitive.ObjectIDFromHex(userId)
	fmt.Println(userId)
	decoder := json.NewDecoder(r.Body)
	var user objects.UpdateUser
	// Decode the update user request from the request body.
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusBadRequest)
		return
	}
	// Create the bson document for the database update request.
	document, err := mg.Document(&user)
	if err != nil {
		http.Error(w, "Can't create DB request", http.StatusInternalServerError)
		return
	}
	fmt.Println(*document)
	// Updates the user into the mongo database.
	response, err := mg.Update(Database, Collection,
		bson.M{"_id": uId}, bson.D{primitive.E{Key: "$set", Value: *document}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(response)
	// Returns the UpdateResult as a json response.
	json.NewEncoder(w).Encode(response)
}

// Handler for authorizing user credentials.
func HandlerAuthorize(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ob objects.AuthorizeRequest
	// Decode the AuthorizeRequest from the request body.
	err := decoder.Decode(&ob)
	if err != nil {
		http.Error(w, "Can't decode the request", http.StatusInternalServerError)
		return
	}
	// Finds the user with the username and the encrypted password.
	response := mg.FindOne(Database, Collection,
		bson.M{"username": ob.Username, "password": Hash(ob.Password)})
	err = response.Err()
	if err != nil {
		http.Error(w, "Invalid login", http.StatusInternalServerError)
		return
	}
	// Decode user information received from the mongo response.
	user, err := DecodeUser(response)
	if err != nil {
		http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		return
	}
	// Generate new tokens for further authentications.
	token, err := GenerateTokens(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Returns the tokens as a json encoded response.
	json.NewEncoder(w).Encode(token)
}

func GenerateTokens(user objects.User) (objects.AuthenticationResponse, error) {
	var token objects.AuthenticationResponse
	// Generate a new access token for the username
	accessToken, err := net.CreateAccessToken(AccessTokenClaims(user))
	if err != nil {
		return token, err
	}
	// Generate a new refresh token for the username
	refreshToken, err := net.CreateRefreshToken(RefreshTokenClaims(user))
	if err != nil {
		return token, err
	}
	// Create the tokens object.
	token.AccessToken = accessToken
	token.RefreshToken = refreshToken
	return token, nil
}

// Generates claims to sign with the access token.
func AccessTokenClaims(user objects.User) (claims jwt.MapClaims) {
	claims = jwt.MapClaims{}
	claims["access"] = true
	claims["username"] = user.Username
	claims["userId"] = user.Id
	claims["name"] = user.Name
	return
}

// Generates claims to sign with the refresh token.
func RefreshTokenClaims(user objects.User) (claims jwt.MapClaims) {
	claims = jwt.MapClaims{}
	claims["refresh"] = true
	claims["name"] = user.Name
	claims["username"] = user.Username
	claims["userId"] = user.Id
	return
}

// Generate the hash string using the SHA256 algorithm for the password.
func Hash(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}

// Handler for generating access token after verifying refresh token.
func HandlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	claims, err := net.Authenticate(r, os.Getenv(net.RefreshClientSecret))
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	userId := claims["userId"]
	uId, _ := primitive.ObjectIDFromHex(fmt.Sprintf("%v", userId))
	// Find the user from the database.
	response := mg.FindOne(Database, Collection, bson.M{"_id": uId})
	err = response.Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Decode user information received from the mongo response.
	user, err := DecodeUser(response)
	if err != nil {
		http.Error(w, "Can't decode the response", http.StatusInternalServerError)
		return
	}
	// Generate new tokens for further authentications.
	token, err := GenerateTokens(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Returns the tokens as a json encoded response.
	json.NewEncoder(w).Encode(token)
}
