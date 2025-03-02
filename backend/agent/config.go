package main

func getDefaultAgent() *Agent {
	return &Agent{ServerURL: "http://127.0.0.1:8000", getEndpoint: "localhost/internal/task",
		sendEndpoint: "localhost/internal/task"}
}
