package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gomodule/redigo/redis"
	"github.com/julienschmidt/httprouter"
)

func GetPokemonWithoutRedis(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// get value from query params
	pokemonName := r.FormValue("pokemon")

	client := http.DefaultClient

	// request to api
	req, err := http.NewRequest("GET", "https://pokeapi.co/api/v2/pokemon/"+pokemonName, nil)
	if err != nil {
		log.Panic(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	bd, _ := ioutil.ReadAll(res.Body)
	w.Write(bd)
}

func GetPokemonWithRedis(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// get value from query params
	pokemonName := r.FormValue("pokemon")

	// setup redis connection
	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Panic(err)
	}

	// check in cache
	reply, err := redis.Bytes(conn.Do("GET", pokemonName))
	if err == nil || pokemonName == "" {
		w.Write(reply)
		return
	}

	// if empty, reqeust to api
	client := http.DefaultClient
	req, err := http.NewRequest("GET", "https://pokeapi.co/api/v2/pokemon/"+pokemonName, nil)
	if err != nil {
		log.Panic(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	bd, _ := ioutil.ReadAll(res.Body)

	// set to cache
	if pokemonName != "" {
		_, err = conn.Do("SET", pokemonName, string(bd))
		if err != nil {
			log.Panic(err)
		}
	}

	// write response
	w.Write(bd)
}
