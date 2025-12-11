package test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorig-node/client/inbound/gnhttp"
	"gorig-node/client/register"
)

// Simulate HTTP calls to verify inbound routing and dispatch are correct.
func TestInboundHTTPInvoke(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svcA := fmt.Sprintf("InboundSvcA_%d", time.Now().UnixNano())
	svcB := fmt.Sprintf("InboundSvcB_%d", time.Now().UnixNano())
	methodA := "Echo"
	methodB := "Ping"

	type echoReq struct {
		Msg string `json:"msg"`
	}
	type echoResp struct {
		Reply string `json:"reply"`
	}
	type pingReq struct {
		Value int `json:"value"`
	}
	type pingResp struct {
		Delta int `json:"delta"`
	}

	//// Invalid function param signature
	//if err := register.Server("BadService").RegName("BadFunc1", func(c *gin.Context, in int) (string, error) {
	//	return "", nil
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// Missing input parameters
	//if err := register.Server("BadService").RegName("BadFunc2", func(c *gin.Context) (string, error) {
	//	return "", nil
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// Missing ctx as first parameter
	//if err := register.Server("BadService").RegName("BadFunc3", func(req1 echoReq, req2 echoReq) (string, error) {
	//	return "", nil
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// Too many return values
	//if err := register.Server("BadService").RegName("BadFunc4", func(c *gin.Context, in echoReq) (string, string, error) {
	//	return "", "", nil
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// Missing expected return value
	//if err := register.Server("BadService").RegName("BadFunc5", func(c *gin.Context, in echoReq) string {
	//	return ""
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// First return value is not a struct
	//if err := register.Server("BadService").RegName("BadFunc6", func(c *gin.Context, in echoReq) (string, error) {
	//	return "", nil
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}
	//
	//// Second return value is not error
	//if err := register.Server("BadService").RegName("BadFunc7", func(c *gin.Context, in echoReq) (echoResp, string) {
	//	return echoResp{}, ""
	//}).Create(); err == nil {
	//	t.Fatalf("register BadService with bad func should have failed")
	//} else {
	//	t.Logf("register BadService with bad func correctly failed: %v", err)
	//}

	if err := register.Server(svcA).RegName(methodA, func(ctx context.Context, req *echoReq) (echoResp, error) {
		logger.Info(ctx, "Received echo request: ", zap.Any("msg", req.Msg))
		return echoResp{Reply: "echo:" + req.Msg}, nil
	}).Create(); err != nil {
		t.Fatalf("register %s failed: %v", svcA, err)
	}
	if err := register.Server(svcB).RegName(methodB, func(ctx context.Context, req pingReq) (pingResp, error) {
		return pingResp{Delta: req.Value + 10}, nil
	}).Create(); err != nil {
		t.Fatalf("register %s failed: %v", svcB, err)
	}

	metaA := register.RegisteredServers()[svcA].MethodMeta[methodA]
	metaB := register.RegisteredServers()[svcB].MethodMeta[methodB]

	engine := gnhttp.NewEngine()

	bodyA, err := register.PackRequest(metaA, []reflect.Value{reflect.ValueOf(&echoReq{Msg: "hello"})})
	if err != nil {
		t.Fatalf("service %s pack request failed: %v", svcA, err)
	}
	respA := performRequest(engine, http.MethodPost, "/"+svcA+"/"+methodA, bodyA)
	if respA.Code != http.StatusOK {
		t.Fatalf("service %s unexpected status %d, body=%s", svcA, respA.Code, respA.Body.String())
	}
	outA, err := register.UnpackResponse(metaA, respA.Body.Bytes())
	if err != nil {
		t.Fatalf("service %s decode failed: %v", svcA, err)
	}
	if errVal := outA[len(outA)-1]; !errVal.IsNil() {
		t.Fatalf("service %s returned error: %v", svcA, errVal.Interface())
	}
	if reply := outA[0].Interface().(echoResp).Reply; reply != "echo:hello" {
		t.Fatalf("service %s unexpected reply: %s", svcA, reply)
	}

	bodyB, err := register.PackRequest(metaB, []reflect.Value{reflect.ValueOf(pingReq{Value: 2})})
	if err != nil {
		t.Fatalf("service %s pack request failed: %v", svcB, err)
	}

	respB := performRequest(engine, http.MethodPost, "/"+svcB+"/"+methodB, bodyB)
	if respB.Code != http.StatusOK {
		t.Fatalf("service %s unexpected status %d, body=%s", svcB, respB.Code, respB.Body.String())
	}
	outB, err := register.UnpackResponse(metaB, respB.Body.Bytes())
	if err != nil {
		t.Fatalf("service %s decode failed: %v", svcB, err)
	}
	if errVal := outB[len(outB)-1]; !errVal.IsNil() {
		t.Fatalf("service %s returned error: %v", svcB, errVal.Interface())
	}
	if delta := outB[0].Interface().(pingResp).Delta; delta != 12 {
		t.Fatalf("service %s unexpected delta: %d", svcB, delta)
	}
}

func performRequest(engine *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("X-Request-ID", "test-request-id")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}
