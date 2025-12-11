package test

//import (
//	"context"
//	"github.com/jom-io/gorig/utils/logger"
//	"github.com/puugy/xzz-go/ordersample"
//	"github.com/puugy/xzz-go/usersample"
//	"testing"
//)

//func TestInvoke(t *testing.T) {
//	ctx := context.Background()
//	ordersample.SetLogger(func(ctx context.Context, msg string) {
//		logger.Info(ctx, msg)
//	}, func(ctx context.Context, msg string) {
//		logger.Error(ctx, msg)
//	})
//	ok, list, err := ordersample.Fallback(func(ctx context.Context) {
//		t.Logf("Invoke ordersample.List fallback")
//	}).List(ctx, ordersample.OrderListReq{
//		UID: "1001",
//	})
//	if err != nil {
//		t.Fatalf("List orders failed: %v", err)
//	}
//	if !ok {
//		t.Fatalf("List orders not ok")
//	} else {
//		t.Logf("List orders success: %+v", list)
//	}
//
//	login, resp, err := usersample.Login(ctx, usersample.LoginReq{
//		Username: "testuser",
//		Password: "password123",
//	})
//
//	if err != nil {
//		t.Fatalf("Login failed: %v", err)
//	}
//	if !login {
//		t.Fatalf("Login not ok")
//	}
//	t.Logf("Login success: %+v", resp)
//}
