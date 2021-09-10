package main

import (
	"database/sql"
	"encoding/json" // package to encode and decode the json into struct and vice versa
	"fmt"
	"log"
	"net/http" // used to access the request and response object of the api
	"strconv"

	"github.com/gomodule/redigo/redis"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq" // postgres golang driver
)

const DB_CONFIG = "user=postgres password=password host=localhost port=5432 dbname=postgres connect_timeout=20 sslmode=disable"

// User model
type User struct {
	UserID   int64  `json:"userid"`
	Name     string `json:"name"`
	Age      int64  `json:"age"`
	Location string `json:"location"`
}

// Response format, Notes: search for omitempty
type Response struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

// Setup connection with postgres db
func SetupConnection() *sql.DB {
	db, err := sql.Open("postgres", DB_CONFIG)
	if err != nil {
		panic(err)
	}

	// Check connection
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	return db
}

// Create user
func CreateUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Set up header, Notes: need to read more on header type
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var user User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Fatalf("Failed to decode request body, %v", err)
	}

	message := "User " + user.Name + " have been inserted!"

	id := InsertUser(user)

	res := Response{
		ID:      id,
		Message: message,
	}

	json.NewEncoder(w).Encode(res)
}

func GetAllUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	users, err := GetAllUsers()

	if err != nil {
		log.Fatalf("Unable to get all user. %v", err)
	}

	// send all the users as response
	json.NewEncoder(w).Encode(users)
}

func GetUserWithoutRedis(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	userid, err := strconv.Atoi(ps.ByName("id"))

	if err != nil {
		log.Fatalf("Invalid user id are given")
	}

	user, err := GetUser(userid)

	if err != nil {
		log.Fatalf("Unable to get user with id:%d. %v", userid, err)
	}

	// Send user as response
	json.NewEncoder(w).Encode(user)
}

func GetUserWithRedis(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	userid, err := strconv.Atoi(ps.ByName("id"))

	if err != nil {
		log.Fatalf("Invalid user id are given")
	}

	user, err := GetUserByRedis(userid)

	if err != nil {
		// Insert new data to redis
		var newuser User

		newuser, err := GetUser(userid)

		if err != nil {
			log.Fatalf("Unable to get user with id:%d. %v", userid, err)
		}

		InsertUserToRedis(newuser)

		user = newuser

		fmt.Println("Fetched from database")
	} else {
		fmt.Println("Fetched from redis")
	}

	// Send user as response
	json.NewEncoder(w).Encode(user)
}

// Insert user to db
func InsertUser(user User) int64 {
	// Setup connection
	db := SetupConnection()
	// Close db on end of function
	defer db.Close()

	// Insert query
	sqlQuery := `INSERT INTO users (name, location, age) VALUES ($1, $2, $3) RETURNING userid`

	var id int64

	// Execute sql query, Notes: Search method for multiple insert
	err := db.QueryRow(sqlQuery, user.Name, user.Location, user.Age).Scan(&id)
	if err != nil {
		log.Fatalf("Unable to execute query, %v", err)
	}

	fmt.Printf("Insertion of user with id:%d is successful\n", id)

	return id
}

func GetAllUsers() ([]User, error) {
	db := SetupConnection()

	defer db.Close()

	var users []User

	// Select query
	sqlQuery := `SELECT * FROM users`

	// Execute sql query
	rows, err := db.Query(sqlQuery)

	if err != nil {
		log.Fatalf("unable to execute query, %v", err)
	}

	// Close statement
	defer rows.Close()

	// Iterate over rows
	for rows.Next() {
		var user User

		// Unmarshal object
		err = rows.Scan(&user.UserID, &user.Name, &user.Age, &user.Location)

		if err != nil {
			log.Fatalf("Unable to get data from the row, %v", err)
		}

		users = append(users, user)
	}

	return users, err
}

func GetUser(userid int) (User, error) {
	db := SetupConnection()

	defer db.Close()

	var user User

	// Select query
	sqlQuery := `SELECT * FROM users WHERE userid = $1`

	// Execute sql query, Notes: Search method for multiple select
	row := db.QueryRow(sqlQuery, userid)

	// Unmarshal the row object to user
	err := row.Scan(&user.UserID, &user.Name, &user.Age, &user.Location)

	switch err {
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
		return user, nil
	case nil:
		return user, nil
	default:
		log.Fatalf("Unable to scan the row. %v", err)
		return user, err
	}
}

func GetUserByRedis(userid int) (User, error) {
	// Setup redis connection
	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	// Close db on end of function
	defer conn.Close()

	var user User

	reply, err := redis.Bytes(conn.Do("GET", userid))
	if err == nil {
		json.Unmarshal([]byte(reply), &user)
		return user, nil
	}

	return user, err
}

func InsertUserToRedis(user User) {
	// Setup redis connection
	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	// Close db on end of function
	defer conn.Close()

	// Set user to JSON format
	user_json, err := json.Marshal(user)
	if err != nil {
		log.Fatalf("Unable to set user with id:%d to JSON. %v", user.UserID, err)
	}

	// Create new value in redis
	_, err = conn.Do("SET", user.UserID, string(user_json))
	if err != nil {
		log.Panic(err)
	}
}
