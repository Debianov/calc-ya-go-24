package main

import "sync"

func main() {
	var (
		wg  sync.WaitGroup
		err error
	)
	//defer func() {
	//	err = exprsList.Read()
	//	if err != nil {
	//		panic(err)
	//	}
	//}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = UpGrpcServer()
		if err != nil {
			panic(err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = UpHttpServer()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()
}
