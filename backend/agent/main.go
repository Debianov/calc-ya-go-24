package main

import (
	"context"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"sync"
	"time"
)

func main() {
	var (
		err          error
		grpcClient   *grpc.ClientConn
		compPowerVar = *backend.CallEnvVarFabric("COMPUTING_POWER", "10")
		wg           sync.WaitGroup
	)

	grpcClient, err = getDefaultGrpcClient()
	if err != nil {
		panic(err)
	}
	var agent = getDefaultAgent(grpcClient)

	var numberCalcGoroutinesInString string
	numberCalcGoroutinesInString, _ = compPowerVar.Get()
	numberCalcGoroutines, err := strconv.ParseInt(numberCalcGoroutinesInString, 10, 32)

	var (
		results          = make(chan *pb.TaskResult, numberCalcGoroutines)
		tasksReadyToCalc = make(chan *pb.TaskToSend, numberCalcGoroutines)
	)

	for range numberCalcGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case task := <-tasksReadyToCalc:
					calcResult, err := Calc(task)
					if err != nil {
						log.Println(err, task.PairId)
					}
					results <- calcResult
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(30 * time.Millisecond):
				task, err := agent.GetTask(context.TODO(), &pb.Empty{})
				if err != nil && task != nil {
					tasksReadyToCalc <- task
				} else if err != nil {
					log.Println(err, task.PairId)
				}
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case result := <-results:
				_, err = agent.SendTask(context.TODO(), result)
				if err != nil {
					log.Println(err, result.PairId)
				}
			}
		}
	}()
	wg.Wait()
}
