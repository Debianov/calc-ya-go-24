package main

func StartHttpServer() (err error) {
	s := GetDefaultHttpServer(getHandler())
	err = s.ListenAndServe()
	return
}
