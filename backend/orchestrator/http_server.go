package main

func UpHttpServer() (err error) {
	s := GetDefaultHttpServer(getHandler())
	err = s.ListenAndServe()
	defer func() {
		err = s.Close()
	}()
	return
}
