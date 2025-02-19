package orchestrator

func StartServer() (err error) {
	s := GetDefaultServer(getHandler())
	err = s.ListenAndServe()
	return
}
