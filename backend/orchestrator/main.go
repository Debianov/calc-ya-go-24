package main

func main() {
	var err error
	err = StartHttpServer()
	if err != nil {
		panic(err)
	}
}
