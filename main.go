package main

import "net/http"

type Config struct {
}

func calcHander(w http.ResponseWriter, r *http.Request) {

}

func main() {
	var err error
	http.HandleFunc("/api/v1/calculate", calcHander)
	s := &http.Server{Addr: "127.0.0.1:8000"}
	err = s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
