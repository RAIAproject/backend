package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

var appid string
var redisClient *redis.Client

func resolveEquation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	equation, found := vars["equation"]
	if !found {
		log.Println("Equation not found")
		w.WriteHeader(400)
		return
	}

	val, err := redisClient.Get(equation).Result()
	if err == nil {
		w.Header().Set("Content-Type", "image/gif")
		w.Write([]byte(val))
		return
	}

	req, err := http.NewRequest("GET", "http://api.wolframalpha.com/v1/simple", nil)

	q := req.URL.Query()
	q.Add("appid", appid)
	q.Add("i", equation)
	req.URL.RawQuery = q.Encode()
	req.URL.RawQuery = strings.Replace(req.URL.RawQuery, "+", "%2B", -1)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Errored when sending request to the server")
		w.WriteHeader(503)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "image/gif")

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}

	_, err = w.Write(bodyBytes)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	bodyString := string(bodyBytes)
	redisClient.Set(equation, bodyString, 0)

}

func main() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})

	appid = os.Getenv("RAIAKEY")
	if len(appid) == 0 {
		log.Fatal("API key not found")
	}
	router := mux.NewRouter().StrictSlash(true)
	router.Path("/question").Queries("equation", "{equation}").HandlerFunc(resolveEquation)
	log.Fatal(http.ListenAndServe(":8080", router))
}
