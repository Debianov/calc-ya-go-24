package main

func StartHttpServer() (err error) {
	s := GetDefaultServer(getHandler())
	err = s.ListenAndServe()
	return
}
