package main

func main() {
	var err error
	err = StartGrpcServer()
	if err != nil {
		panic(err)
	}
	err = StartHttpServer()
	if err != nil {
		panic(err)
	}
}
