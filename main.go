package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()
	router.GET("/user", GetAllUser)
	router.GET("/user/:id", GetUserWithoutRedis)
	router.GET("/user-redis/:id", GetUserWithRedis)
	router.POST("/user", CreateUser)
	router.GET("/pokemonwithredis", GetPokemonWithRedis)
	router.GET("/pokemonwithoutredis", GetPokemonWithoutRedis)
	log.Fatal(http.ListenAndServe(":3001", router))
}
