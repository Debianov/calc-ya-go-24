package backend

import (
	"fmt"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"time"
)

type TimeoutExecution struct {
	operationTime time.Duration
	factTime      time.Duration
	operation     string
	pairId        int
}

func (t TimeoutExecution) Error() string {
	exprId, taskId := pkg.Unpair(t.pairId)
	return fmt.Sprintf("возник timeout при обработке task: %d из expression %d, оператор: %s; время на "+
		"выполнение: %s,фактически: %s", taskId, exprId, t.operation, t.operationTime, t.factTime)
}

type TaskIDNotExist struct {
	taskId int
}

func (t TaskIDNotExist) Error() string {
	return fmt.Sprintf("задачи с ID %d не найдена", t.taskId)
}

var NoReadyTaskError = &NoReadyTask{}

type NoReadyTask struct {
}

func (n NoReadyTask) Error() string {
	return fmt.Sprintln("нет готовых задач на данный момент")
}

type OneReadyTaskBug struct {
	ExprId int
}

func (n OneReadyTaskBug) Error() string {
	return fmt.Sprintf("(bug) разработчиком ожидается, что выданный expr (id %d) будет иметь хотя бы 1 готовый"+
		" к отправке task.\n", n.ExprId)
}
