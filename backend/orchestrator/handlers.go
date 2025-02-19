package orchestrator

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"io"
	"log"
	"net/http"
	"strconv"
)

//var expressionsList = make([]*backend.Expression, 0)

// var expr = backend.ExpressionFabric(postfix, expressionsList)
// expr.DivideToParallelise()
// expressionsList = append(expressionsList, expr)
//type expressionsQueue struct {
//	queue []*backend.Expression
//}

func calcHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)
	if r.Method != http.MethodPost {
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return
	}
	var (
		buf           []byte
		requestStruct backend.RequestJson
		reader        io.ReadCloser
	)
	reader = r.Body
	buf, err = io.ReadAll(reader)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(buf, &requestStruct)
	if err != nil {
		log.Panic(err)
	}
	ok, postfix := pkg.GeneratePostfix(requestStruct.Expression)
	if !ok {
		w.WriteHeader(422)
		return
	}
	id, err := expressionsList.push(postfix)
	expr, _ := expressionsList.get(id)
	marshaledExpr, err := expr.MarshalID()
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(marshaledExpr)
	if err != nil {
		log.Panic(err)
	}
	w.WriteHeader(201)
}

func expressionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	var err error
	exprs := expressionsList.getAllExprs()
	var exprsJsonHandler = struct {
		Expressions []backend.Expression `json:"expressions"`
	}{exprs}
	exprsHandlerInBytes, err := json.Marshal(&exprsJsonHandler)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(exprsHandlerInBytes)
	if err != nil {
		log.Panic(err)
	}
}

func expressionIdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	var err error
	id := r.PathValue("ID")
	idInINt, err := strconv.ParseInt(id)
	if err != nil {
		log.Panic(err)
	}
	expr, exist := expressionsList.get(idInINt)
	if !exist {
		w.WriteHeader(404)
		return
	}
	var exprJsonHandler = struct {
		ExprInstance backend.Expression `json:"expression"`
	}{expr}
	exprHandlerInBytes, err := json.Marshal(&exprJsonHandler)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(exprHandlerInBytes)
	if err != nil {
		log.Panic(err)
	}
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		taskGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		taskPostHandler(w, r)
	}
}

func taskGetHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	expr, exist := expressionsList.getFreeExpr()
	if !exist {
		w.WriteHeader(404)
		return
	}
	task, exist := expr.getFreeTask()
	if !exist {
		w.WriteHeader(404)
		return
	}
	var taskJsonHandler = struct {
		Task backend.Task `json:"task"`
	}{task}
	taskJsonHandlerInBytes, err := json.Marshal(&taskJsonHandler)
	if err != nil {
		log.Panic(err)
	}
	w.Write(taskJsonHandlerInBytes)
}

func taskPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		return
	}
	var (
		err    error
		reqBuf = make([]byte, r.ContentLength)
	)
	_, err = r.Body.Read(reqBuf)
	if err != nil {
		log.Panic(err)
	}
	var (
		reqInJson backend.TaskIsDone
	)
	err = json.Unmarshal(reqBuf, &reqInJson)
	if err != nil {
		log.Panic(err)
	}
	task, exist := agentTasks.pop(reqInJson.ID)
	if !exist {
		w.WriteHeader(404)
		return
	}
	err = task.writeResult(reqInJson.Result) // TODO гарантировать, что операция будет выполнена только один раз
	// иначе ошибка
	if err != nil {
		log.Panic(err) // TODO err: BUG: разработчиком ожидается, что результат одной и той же задачи не может быть записан больше одног раза
	}
	var expr = task.Expression
	expr.updateTask(task)
}

func panicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("response %s, status code: 500", w)
				writeInternalServerError(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeInternalServerError(w http.ResponseWriter) {
	var (
		buf         []byte
		err         error
		errResponse = &backend.ErrorJson{Error: "Internal server error"}
	)
	buf, err = errResponse.Marshal()
	if err != nil {
		log.Panic(err)
	}
	w.WriteHeader(500)
	_, err = w.Write(buf)
	if err != nil {
		log.Panic(err)
	}
	return
}

func getHandler() (handler http.Handler) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", calcHandler)
	mux.HandleFunc("/api/v1/expressions", expressionsHandler)
	mux.HandleFunc("/api/v1/expressions/{ID}", expressionIdHandler)
	mux.HandleFunc("/internal/task", taskHandler)
	handler = panicMiddleware(mux)
	return
}
