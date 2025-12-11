package register

import (
	"context"
	"errors"
	"fmt"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
	"gorig-node/gncfg"
	"gorig-node/internal/utils"
	"reflect"
	"regexp"
	"sync"
)

type ServerName = string

// only letters, numbers, underscore; must start with letter
const serverNamePattern = "^[a-zA-Z][a-zA-Z0-9_]*$"

type MethodMeta struct {
	FnValue reflect.Value
	FnType  reflect.Type

	HasCtx  bool
	CtxType reflect.Type

	InTypes  []reflect.Type // includes ctx
	OutTypes []reflect.Type // includes error

	ArgNames []string // inferred + user supplied
}

type ServerRegister struct {
	ServiceName ServerName
	Host        string
	Version     string
	Environment string
	Apis        []ApiInfo `json:"apis"`
	FnMap       map[string]reflect.Value
	MethodMeta  map[string]MethodMeta `json:"-"`
	created     bool                  // whether Create() has been called
}

type ServerCreator struct {
	srv   *ServerRegister
	error error
}

var (
	registeredServers = sync.Map{} // map[string]*serverRegister
)

func RegisteredServers() map[ServerName]*ServerRegister {
	result := make(map[string]*ServerRegister)
	registeredServers.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*ServerRegister)
		return true
	})
	return result
}

// Server("user")
func Server(name ServerName) *ServerCreator {
	creator := &ServerCreator{}

	if val, ok := registeredServers.Load(name); ok {
		creator.srv = val.(*ServerRegister)
		return creator
	}

	// new serverRegister
	reg := &ServerRegister{
		ServiceName: name,
		Apis:        make([]ApiInfo, 0),
		FnMap:       make(map[string]reflect.Value),
		MethodMeta:  map[string]MethodMeta{},
		created:     false,
	}
	registeredServers.Store(name, reg)
	creator.srv = reg
	return creator
}

// Create() validates service name and confirms registration
func (s *ServerCreator) Create() error {
	if s.error != nil {
		logger.Error(context.Background(), fmt.Sprintf("register service %s failed: %v", s.srv.ServiceName, s.error))
		return s.error
	}

	matched, _ := regexp.MatchString(serverNamePattern, s.srv.ServiceName)
	if !matched {
		return fmt.Errorf("invalid service name: %s", s.srv.ServiceName)
	}

	s.srv.created = true
	registeredServers.Store(s.srv.ServiceName, s.srv)
	return nil
}

// Host("10.0.0.5:8081")
func (s *ServerCreator) Host(h string) *ServerCreator {
	if s.srv != nil {
		s.srv.Host = h
	}
	return s
}

// Version("v1.0.0")
func (s *ServerCreator) Version(v string) *ServerCreator {
	if s.srv != nil {
		s.srv.Version = v
		// 还未注册的方法会在 RegName 时继承
		for i := range s.srv.Apis {
			s.srv.Apis[i].Version = v
		}
	}
	return s
}

// Env sets deployment environment; used for branch selection when generating SDK.
func (s *ServerCreator) Env(env string) *ServerCreator {
	if s.srv != nil {
		s.srv.Environment = env
		for i := range s.srv.Apis {
			s.srv.Apis[i].Environment = env
		}
	}
	return s
}

func (c *ServerCreator) Reg(fn interface{}) *ServerCreator {
	name := utils.GetFuncName(fn)
	return c.RegName(name, fn)
}

func (c *ServerCreator) RegName(name string, fn interface{}, argNames ...string) *ServerCreator {
	meta, api, err := makeWrapper(c.srv.ServiceName, name, fn, argNames)
	if err != nil {
		c.error = err
		return c
	}

	// 继承服务级版本
	api.Version = c.srv.Version
	api.Environment = c.srv.Environment

	c.srv.FnMap[name] = meta.FnValue
	c.srv.MethodMeta[name] = meta
	c.srv.Apis = append(c.srv.Apis, api)

	return c
}

// Start() reports all services that have successfully called Create()
func Start() error {
	if gncfg.Cfg.HubAddr == "" {
		return errors.New("HubAddr not set via UseConfig")
	}

	localIP := autoDetectIP()
	if localIP == "" && gncfg.Cfg.NodeAddr == "" {
		return errors.New("failed to detect local IP; ensure network is OK or set Host manually")
	}

	var firstErr error
	var hasRegistered bool
	registeredServers.Range(func(name, value interface{}) bool {
		srv := value.(*ServerRegister)

		// Skip if Create not called
		if !srv.created {
			return true
		}

		// Auto-fill host
		if srv.Host == "" {
			if gncfg.Cfg.NodeAddr != "" {
				srv.Host = gncfg.Cfg.NodeAddr
			} else {
				srv.Host = localIP + gncfg.DefNodePort
			}
		}

		if err := sendRegisterWithTimeout(gncfg.Cfg.HubAddr, srv); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			logger.Error(context.Background(), "report to registry failed", zap.String("service", srv.ServiceName), zap.String("hub", gncfg.Cfg.HubAddr), zap.Error(err))
			return true
		}

		hasRegistered = true
		logger.Info(context.Background(), "report to registry succeeded", zap.String("service", srv.ServiceName), zap.String("hub", gncfg.Cfg.HubAddr), zap.String("host", srv.Host))
		return true
	})

	if hasRegistered {
		startHeartbeatLoop(gncfg.Cfg.HubAddr)
	}

	return firstErr
}
