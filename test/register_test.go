package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jom-io/gorig-node/client/register"
	"github.com/jom-io/gorig-node/gncfg"
)

// Simulate real flow: register two services and finally call Start.
func TestRegisterFlow(t *testing.T) {
	ip := detectPrivateIP()
	if ip == "" {
		t.Skip("no private IP available; cannot simulate full Start flow")
	}

	type recordedReq struct {
		Path string
		Body []byte
	}

	type regPayload struct {
		Service       string             `json:"service"`
		Host          string             `json:"host"`
		Apis          []register.ApiInfo `json:"apis"`
		ServiceLegacy string             `json:"ServiceName"`
		HostLegacy    string             `json:"Host"`
	}

	reqCh := make(chan recordedReq, 16)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqCh <- recordedReq{Path: r.URL.Path, Body: body}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer func() {
		register.Stop()
		ts.Close()
	}()

	gncfg.UseConfig(gncfg.GlobalConfig{
		HubAddr:  ts.URL,
		NodeAddr: ip + gncfg.DefNodePort,
	})

	userService := fmt.Sprintf("UserService_%d", time.Now().UnixNano())
	orderService := fmt.Sprintf("OrderService_%d", time.Now().UnixNano())

	// User service: Host not set, expect Start to auto-fill.
	type loginReq struct {
		User string
	}
	type loginResp struct {
		Token string
	}
	loginHandler := func(ctx context.Context, req loginReq) (loginResp, error) {
		return loginResp{Token: "tk-" + req.User}, nil
	}
	if err := register.Server(userService).RegName("Login", loginHandler).Create(); err != nil {
		t.Fatalf("register user service failed: %v", err)
	}

	// Order service: custom Host.
	type orderReq struct {
		IDs []int
	}
	type orderResp struct {
		Count int
	}
	listOrderHandler := func(ctx context.Context, req orderReq) (orderResp, error) {
		return orderResp{Count: len(req.IDs)}, nil
	}
	if err := register.Server(orderService).Host("10.0.0.5:8081").RegName("ListOrders", listOrderHandler).Create(); err != nil {
		t.Fatalf("register order service failed: %v", err)
	}

	// Invalid signature (user misuse simulation).
	badFn := func(bad string, ctx context.Context, in string) string { return in }
	if err := register.Server("BadService").RegName("BadFunc", badFn).Create(); err == nil {
		t.Fatalf("invalid signature should fail")
	} else {
		t.Logf("invalid signature correctly failed: %v", err)
	}

	// Invalid returns (user misuse). Only one error allowed and must be last.
	badFn2 := func(ctx context.Context, in string) (error, error) { return nil, nil }
	if err := register.Server("BadService2").RegName("BadFunc2", badFn2).Create(); err == nil {
		t.Fatalf("invalid returns should fail")
	} else {
		t.Logf("invalid returns correctly failed: %v", err)
	}

	badFn3 := func(ctx context.Context, in string) (int, error, string) { return 0, nil, "" }
	if err := register.Server("BadService3").RegName("BadFunc3", badFn3).Create(); err == nil {
		t.Fatalf("invalid returns should fail")
	} else {
		t.Logf("invalid returns correctly failed: %v", err)
	}

	// Finally start: simulate reporting to registry.
	if err := register.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	regBodies := map[string]regPayload{}
	var heartbeatReq *recordedReq
	timeout := time.After(2 * time.Second)
	hasAllRegisters := func() bool {
		_, hasUser := regBodies[userService]
		_, hasOrder := regBodies[orderService]
		return hasUser && hasOrder
	}

	for !hasAllRegisters() || heartbeatReq == nil {
		select {
		case req := <-reqCh:
			switch req.Path {
			case "/register":
				var body regPayload
				if err := json.Unmarshal(req.Body, &body); err != nil {
					t.Fatalf("failed to parse register payload: %v", err)
				}
				svc := body.Service
				if svc == "" {
					svc = body.ServiceLegacy
				}
				body.Host = firstNonEmpty(body.Host, body.HostLegacy)
				regBodies[svc] = body
			case "/heartbeat":
				if heartbeatReq == nil {
					heartbeatReq = &req
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for register/heartbeat requests, register=%d heartbeat=%t", len(regBodies), heartbeatReq != nil)
		}
	}

	if userReg, ok := regBodies[userService]; !ok {
		t.Fatalf("missing register request for %s", userService)
	} else {
		if userReg.Host != gncfg.Cfg.NodeAddr {
			t.Fatalf("user service host mismatch: %s", userReg.Host)
		}
		if len(userReg.Apis) != 1 {
			t.Fatalf("user service api count mismatch: %d", len(userReg.Apis))
		}
	}

	if orderReg, ok := regBodies[orderService]; !ok {
		t.Fatalf("missing register request for %s", orderService)
	} else {
		if orderReg.Host != "10.0.0.5:8081" {
			t.Fatalf("order service host mismatch: %s", orderReg.Host)
		}
		if len(orderReg.Apis) != 1 {
			t.Fatalf("order service api count mismatch: %d", len(orderReg.Apis))
		}
	}

	type hbPayload struct {
		Services []struct {
			Service string `json:"service"`
			Host    string `json:"host"`
		} `json:"services"`
	}

	var hbBody hbPayload
	if err := json.Unmarshal(heartbeatReq.Body, &hbBody); err != nil {
		t.Fatalf("failed to parse heartbeat payload: %v", err)
	}

	hbBodies := map[string]string{}
	for _, hb := range hbBody.Services {
		hbBodies[hb.Service] = hb.Host
	}

	if host, ok := hbBodies[userService]; !ok {
		t.Fatalf("missing heartbeat for %s", userService)
	} else if host != gncfg.Cfg.NodeAddr {
		t.Fatalf("user service heartbeat host mismatch: %s", host)
	}

	if host, ok := hbBodies[orderService]; !ok {
		t.Fatalf("missing heartbeat for %s", orderService)
	} else if host != "10.0.0.5:8081" {
		t.Fatalf("order service heartbeat host mismatch: %s", host)
	}
}

// Copy the auto-detect IP logic to check if prerequisites are satisfied.
func detectPrivateIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP == nil {
			continue
		}
		if ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		ip := ipnet.IP.String()
		if strings.HasPrefix(ip, "172.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
			return ip
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
