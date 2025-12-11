package register

import (
	"bytes"
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type registerRequest struct {
	Service string    `json:"service"`
	Version string    `json:"version,omitempty"`
	Env     string    `json:"env,omitempty"`
	Host    string    `json:"host"`
	Apis    []ApiInfo `json:"apis"`
	// legacy for hub compatibility (kept for a while)
	ServiceLegacy string `json:"ServiceName,omitempty"`
	VersionLegacy string `json:"Version,omitempty"`
	EnvLegacy     string `json:"Environment,omitempty"`
	HostLegacy    string `json:"Host,omitempty"`
}

type heartbeatRequest struct {
	Service string `json:"service"`
	Host    string `json:"host"`
}

type heartbeatBatchRequest struct {
	Services []heartbeatRequest `json:"services"`
}

var (
	requestTimeout    = 5 * time.Second
	heartbeatInterval = time.Minute
	httpClient        = &http.Client{Timeout: requestTimeout}
	heartbeatMu       sync.Mutex
	heartbeatCancel   context.CancelFunc
	heartbeatRunning  bool
	enableHBLog       bool
)

func sendRegisterWithTimeout(hubAddr string, srv *ServerRegister) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return sendRegister(ctx, hubAddr, srv)
}

func sendRegister(ctx context.Context, hubAddr string, srv *ServerRegister) error {
	url := buildHubURL(hubAddr, "/register")
	payload := registerRequest{
		Service:       srv.ServiceName,
		Version:       srv.Version,
		Env:           srv.Environment,
		Host:          srv.Host,
		Apis:          srv.Apis,
		ServiceLegacy: srv.ServiceName,
		VersionLegacy: srv.Version,
		EnvLegacy:     srv.Environment,
		HostLegacy:    srv.Host,
	}
	return postJSON(ctx, url, payload)
}

func sendHeartbeatBatchWithTimeout(hubAddr string, batch heartbeatBatchRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return sendHeartbeatBatch(ctx, hubAddr, batch)
}

func sendHeartbeatBatch(ctx context.Context, hubAddr string, batch heartbeatBatchRequest) error {
	if len(batch.Services) == 0 {
		return nil
	}
	url := buildHubURL(hubAddr, "/heartbeat")
	return postJSON(ctx, url, batch)
}

func startHeartbeatLoop(hubAddr string) {
	heartbeatMu.Lock()
	if heartbeatRunning {
		heartbeatMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	heartbeatRunning = true
	heartbeatCancel = cancel
	heartbeatMu.Unlock()

	go func() {
		defer func() {
			heartbeatMu.Lock()
			heartbeatRunning = false
			heartbeatCancel = nil
			heartbeatMu.Unlock()
		}()

		sendHeartbeatsOnce(hubAddr)

		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendHeartbeatsOnce(hubAddr)
			}
		}
	}()
}

func Stop() {
	heartbeatMu.Lock()
	defer heartbeatMu.Unlock()
	if heartbeatCancel != nil {
		heartbeatCancel()
	}
	heartbeatRunning = false
	heartbeatCancel = nil
}

// EnableHeartbeatLog controls whether heartbeat success logs are printed (disabled by default).
func EnableHeartbeatLog(enable bool) {
	enableHBLog = enable
}

func buildHubURL(base, path string) string {
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func postJSON(ctx context.Context, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("request %s failed with status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// sendHeartbeatsOnce sends heartbeats for all created services.
func sendHeartbeatsOnce(hubAddr string) {
	var (
		batch   heartbeatBatchRequest
		records []string
	)
	registeredServers.Range(func(_, value interface{}) bool {
		srv := value.(*ServerRegister)
		if !srv.created || srv.Host == "" {
			return true
		}
		batch.Services = append(batch.Services, heartbeatRequest{
			Service: srv.ServiceName,
			Host:    srv.Host,
		})
		records = append(records, fmt.Sprintf("%s@%s", srv.ServiceName, srv.Host))
		return true
	})

	if len(batch.Services) == 0 {
		return
	}

	if err := sendHeartbeatBatchWithTimeout(hubAddr, batch); err != nil {
		logger.Error(context.Background(), "heartbeat failed", zap.String("hub", hubAddr), zap.Strings("services", records), zap.Error(err))
		return
	}
	if enableHBLog {
		logger.Info(context.Background(), "heartbeat succeeded", zap.String("hub", hubAddr), zap.Strings("services", records))
	}
}
