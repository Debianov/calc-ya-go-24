package backend

import (
	"slices"
	"sync"
	"time"
)

// Tasks  обёртка над pkg.Stack с дополнительными методами. Нужен для обработки случаев, когда
// несколько Task-ов готовы и нужно продолжить работу других Task-ов, зависящие от первых.
// Task-и, которые нужны для третьего, пока переходят в отдельный tasksWithResultStack.
// В случае, когда все необходимые Task-и обновлены, они удаляются из stackWithUpdatedTasks
// , а их результаты записываются в ожидающий Task (первый Task из buf).
// Для работы с TaskToSend встроена структура.
type Tasks struct {
	sentTasks
	buf                                []Task
	tasksCountBeforeWaitingTask        int
	updatedTasksCountBeforeWaitingTask int
}

func (t *Tasks) add(task Task) {
	t.buf = append(t.buf, task)
}

func (t *Tasks) Len() int {
	return len(t.buf)
}

func (t *Tasks) getFirst() (task *Task) {
	task = &t.buf[t.tasksCountBeforeWaitingTask]
	if !t.isWaiting(task) {
		t.tasksCountBeforeWaitingTask++
		return
	} else {
		var expectedTask Task
		if t.updatedTasksCountBeforeWaitingTask == t.tasksCountBeforeWaitingTask { // цикл в
			// горутине не требуется, поскольку агент будут самостоятельно тыкать в сервер, чтоб тот проверил на
			// наличие свободных таск
			if task.Arg2 != nil {
				expectedTask = t.buf[0]
				slices.Delete(t.buf, 0, 1)
				task.Arg2 = int64(expectedTask.result)
			}
			if task.Arg1 != nil {
				expectedTask = t.buf[0]
				slices.Delete(t.buf, 0, 1)
				task.Arg1 = int64(expectedTask.result)
			}
			task.isReadyToCalculation = true
			t.updatedTasksCountBeforeWaitingTask = 0
			t.tasksCountBeforeWaitingTask = 0
		}
		return
	}
}

func (t *Tasks) isWaiting(task *Task) bool {
	return task.Arg1 == nil && task.Arg2 == nil
}

func (t *Tasks) CountUpdatedTask() {
	t.updatedTasksCountBeforeWaitingTask++
}

// sentTasks — map для работы с TaskToSend структурой.
type sentTasks struct {
	buf map[int]TaskToSend
	mut sync.Mutex
}

func (t *sentTasks) fabricAppendInSentTasks(readyTask *Task, timeAtSendingTask time.Time) (result TaskToSend) {
	result = TaskToSend{
		Task:              readyTask,
		timeAtSendingTask: timeAtSendingTask,
	}
	t.mut.Lock()
	t.buf[readyTask.PairID] = result
	t.mut.Unlock()
	return
}

func (t *sentTasks) getTask(taskId int) (*Task, time.Time, bool) {
	t.mut.Lock()
	taskWithTimer, ok := t.buf[taskId]
	if ok {
		delete(t.buf, taskId)
	}
	t.mut.Unlock()
	return taskWithTimer.Task, taskWithTimer.timeAtSendingTask, ok
}

func sentTasksFabric() *sentTasks {
	return &sentTasks{
		buf: make(map[int]TaskToSend),
	}
}

func TaskSpaceFabric() *Tasks {
	newSentTasks := *sentTasksFabric()
	return &Tasks{sentTasks: newSentTasks}
}
