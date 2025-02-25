package backend

import (
	"fmt"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"time"
)

type timeoutExecution struct {
	operationTime time.Duration
	factTime      time.Duration
	operation     string
	pairId        int
}

func (t timeoutExecution) Error() string {
	exprId, taskId := pkg.Unpair(t.pairId)
	return fmt.Sprintf("возник timeout при обработке task: %d из expression %d, оператор: %s, время на выполнение: %s,"+
		"фактически: %s", taskId, exprId, t.operation, t.operationTime, t.factTime)
}

type taskIDNotExist struct {
	taskId int
}

func (t taskIDNotExist) Error() string {
	return fmt.Sprintf("задачи с ID %d не найдена", t.taskId)
}
