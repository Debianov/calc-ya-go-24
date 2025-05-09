package backend

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

const DefaultHttpServerUrl = "http://127.0.0.1:8000"

func loginUser(t *testing.T, creds CommonUser) string {
	var (
		reqBuf  []byte
		resp    *http.Response
		respBuf []byte
		token   JwtTokenJsonWrapperStub
		err     error
	)
	reqBuf, err = json.Marshal(&creds)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(DefaultHttpServerUrl+"/api/v1/login", "application/json", bytes.NewReader(reqBuf))
	if err != nil {
		t.Fatal(err)
	}
	respBuf, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(respBuf, &token)
	if err != nil {
		t.Fatal(err)
	}
	return token.Token
}

var (
	registeredUser CommonUser = &UserStub{
		Login:    "test",
		Password: "qwerty",
		Id:       0,
	}
)

func checkExpressions(t *testing.T) {
	var token = loginUser(t, registeredUser)
	var (
		requestsToCalc = []*RequestJsonStub{{Token: token, Expression: "2+2*4"}, {Token: token,
			Expression: "32+(4*2)/4"}, {Token: token, Expression: "2*3+4*10"}}
		realCalcResponses     []IdHolder
		expectedCalcResponses = []IdHolder{{0}, {1}, {2}}
	)
	realCalcResponses = callCalcApi[*RequestJsonStub](t, requestsToCalc)
	assert.ElementsMatch(t, expectedCalcResponses, realCalcResponses)
	var (
		requestsToExprs        = JwtTokenJsonWrapperStub{Token: token}
		realExprsResponses     []ExpressionStub
		expectedExprsResponses = []ExpressionStub{{Id: 0, Status: Completed, Result: 10},
			{Id: 1, Status: Completed, Result: 34},
			{Id: 2, Status: Completed, Result: 46}}
	)
	time.Sleep(1 * time.Second) // примерное время на передачу туда-обратно и обработку
	realExprsResponses = callExpressionsApi(t, &requestsToExprs)
	assert.ElementsMatch(t, expectedExprsResponses, realExprsResponses)
}

type IdHolder struct {
	Id int `json:"id"`
}

func (i *IdHolder) Marshal() (result []byte, err error) {
	return json.Marshal(i)
}

func callCalcApi[T JsonPayload](t *testing.T, requestsToCalc []T) (result []IdHolder) {
	var (
		jsonBuf     []byte
		err         error
		resp        *http.Response
		respBuf     []byte
		resultEntry IdHolder
	)
	for _, req := range requestsToCalc {
		jsonBuf, err = req.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.Post(DefaultHttpServerUrl+"/api/v1/calculate", "application/json",
			bytes.NewReader(jsonBuf))
		respBuf, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(respBuf, &resultEntry)
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, resultEntry)
	}
	return
}

func callExpressionsApi(t *testing.T, requestsToExprs JsonPayload) (result []ExpressionStub) {
	var (
		err         error
		reqBuf      []byte
		resp        *http.Response
		respBuf     []byte
		resultEntry = ExpressionsJsonTitleStub{make([]ExpressionStub, 0)}
	)
	reqBuf, err = requestsToExprs.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(DefaultHttpServerUrl+"/api/v1/expressions", "application/json",
		bytes.NewReader(reqBuf))
	if err != nil {
		t.Fatal(err)
	}
	respBuf, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(respBuf, &resultEntry)
	if err != nil {
		t.Fatal(err)
	}
	result = resultEntry.Expressions
	return
}

func checkStateSaving(t *testing.T) {
	var token = loginUser(t, registeredUser)
	var (
		requestsToExprs        = JwtTokenJsonWrapperStub{Token: token}
		realExprsResponses     []ExpressionStub
		expectedExprsResponses = []ExpressionStub{{Id: 0, Status: Completed, Result: 10},
			{Id: 1, Status: Completed, Result: 34},
			{Id: 2, Status: Completed, Result: 46}}
	)
	realExprsResponses = callExpressionsApi(t, &requestsToExprs)
	assert.ElementsMatch(t, expectedExprsResponses, realExprsResponses)
}

func TestIntegration(t *testing.T) {
	var (
		err          error
		stopServices func()
	)
	stopServices, err = prepareServices()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("checkExpressions", checkExpressions)
	stopServices()

	stopServices, err = prepareServices()
	if err != nil {
		t.Fatal(err)
	}
	defer stopServices()
	t.Run("checkStateSaving", checkStateSaving)
}

func prepareServices() (stopFn func(), err error) {
	var (
		orchCmd  *exec.Cmd
		agentCmd *exec.Cmd
	)
	orchCmd, err = startOrchestrator()
	if err != nil {
		return nil, err
	}
	agentCmd, err = startAgent()
	if err != nil {
		return nil, err
	}
	stopFn = func() {
		orchPgid, _ := syscall.Getpgid(orchCmd.Process.Pid)
		agentPgid, _ := syscall.Getpgid(agentCmd.Process.Pid)
		syscall.Kill(-orchPgid, syscall.SIGINT) // если указан отрицательный PGID, то сигнал отправляется всем процессам
		// с данным PGID
		syscall.Kill(-agentPgid, syscall.SIGINT)
	}
	time.Sleep(3 * time.Second) // процессы не успевают подняться
	// TODO:
	return
}

func startOrchestrator() (cmd *exec.Cmd, err error) {
	cmd = exec.Command("go", "run", "github.com/Debianov/calc-ya-go-24/backend/orchestrator")
	cmd.Dir = "./orchestrator"
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // все дочерние процессы под один общий PID
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, err
}

func startAgent() (cmd *exec.Cmd, err error) {
	cmd = exec.Command("go", "run", "github.com/Debianov/calc-ya-go-24/backend/agent")
	cmd.Dir = "./agent"
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // все дочерние процессы под один общий групповой PID
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, err
}
