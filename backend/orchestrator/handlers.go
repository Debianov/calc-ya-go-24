package orchestrator

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"io"
	"log"
	"maps"
	"net/http"
	"strconv"
	"sync"
)

//var expressionsList = make([]*backend.Expression, 0)

// var expr = backend.ExpressionFabric(postfix, expressionsList)
// expr.DivideIntoTasks()
// expressionsList = append(expressionsList, expr)
//type expressionsQueue struct {
//	queue []*backend.Expression
//}

type expressionsList struct {
	mut   sync.Mutex
	exprs map[int]*backend.Expression
}

func (e *expressionsList) FabricPush(postfix []string) (newExpr *backend.Expression, newId int) {
	newId = e.generateId()
	newExpr = &backend.Expression{Postfix: postfix, ID: newId, Status: backend.Ready} // TODO init стека, нужна фабрика
	newExpr.DivideIntoTasks()
	e.mut.Lock()
	e.exprs[newId] = newExpr
	e.mut.Unlock()
	return
}

func (e *expressionsList) generateId() (id int) {
	e.mut.Lock()
	defer e.mut.Unlock()
	return len(e.exprs)
}

func (e *expressionsList) GetAllExprs() []*backend.Expression {
	e.mut.Lock()
	defer e.mut.Unlock()
	var exprs interface{}
	exprs = maps.Values(e.exprs)
	return exprs.([]*backend.Expression)
}

func (e *expressionsList) Get(id int) (*backend.Expression, bool) {
	e.mut.Lock()
	var result, ok = e.exprs[id]
	e.mut.Unlock()
	return result, ok
}

func (e *expressionsList) getReadyExpr() (expr *backend.Expression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	for _, v := range e.exprs {
		if v.Status == backend.Ready {
			return v
		}
	}
	return nil
}

var exprsList = &expressionsList{}

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
	postfix, ok := pkg.GeneratePostfix(requestStruct.Expression)
	if !ok {
		w.WriteHeader(422)
		return
	}
	expr, _ := exprsList.FabricPush(postfix)
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
	exprs := exprsList.GetAllExprs()
	var exprsJsonHandler = struct {
		Expressions []*backend.Expression `json:"expressions"`
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
	idInINt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Panic(err)
	}
	expr, exist := exprsList.Get(int(idInINt))
	if !exist {
		w.WriteHeader(404)
		return
	}
	var exprJsonHandler = struct {
		ExprInstance backend.Expression `json:"expression"`
	}{*expr}
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
	expr := exprsList.getReadyExpr()
	if expr == nil {
		w.WriteHeader(404)
		return
	}
	task := expr.GetReadyToSendTask()
	var taskJsonHandler = struct {
		Task backend.Task `json:"task"`
	}{*task}
	taskJsonHandlerInBytes, err := json.Marshal(&taskJsonHandler)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(taskJsonHandlerInBytes)
	if err != nil {
		log.Panic(err)
	}
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
		reqInJson = struct {
			ID     int `json:"ID"`
			Result int `json:"result"`
		}{}
	)
	err = json.Unmarshal(reqBuf, &reqInJson)
	if err != nil {
		log.Panic(err)
	}
	exprID, taskID := pkg.Unpair(reqInJson.ID)
	expr, ok := exprsList.Get(exprID)
	if !ok {
		w.WriteHeader(404)
		return
	}
	task, ok := expr.GetTask(taskID)
	if !ok {
		w.WriteHeader(404)
		return
	}
	err = task.writeResult(reqInJson.Result) // TODO гарантировать, что операция будет выполнена только один раз
	// TODO иначе ошибка
	if err != nil {
		log.Panic(err) // TODO err: BUG: разработчиком ожидается, что результат одной и той же задачи не может быть
		// записан больше одног раза
	}
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
