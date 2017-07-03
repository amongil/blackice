package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/amongil/blackice/blackice/ec2utils"
	"github.com/julienschmidt/httprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func instances(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cl := ec2utils.NewClient("eu-central-1")
	instances, err := cl.GetInstancesByKeyPair(ps.ByName("keyname"))
	if err != nil {
		fmt.Println(err.Error())
	}
	// Convert our structures to json
	res, err := json.MarshalIndent(instances, "", "\t")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, string(res))
}

// New creates a new http server that can be started and stopped
func New() *http.Server {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/hello/:name", hello)
	router.GET("/instances/:keyname", instances)
	addr := "127.0.0.1:8080"
	srv := &http.Server{
		Handler: router,
		Addr:    addr,
	}
	return srv
}

// Start makes the server start listening for requests
func Start(srv *http.Server) {
	log.Fatal(srv.ListenAndServe())
}

// Stop makes the server stop listening for requests
func Stop(srv *http.Server) {
	srv.Shutdown(nil)
}
