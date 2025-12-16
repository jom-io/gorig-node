package main

import (
	"context"
	"fmt"

	"github.com/jom-io/gorig-node/client/register"
)

// Register a few sample services before main starts for local demo.
// Replace with real business services as needed.
func init() {
	// Enable this to see heartbeat success logs (disabled by default).
	//register.EnableHeartbeatLog(true)

	//_ = register.Server("UserSample").
	//	Reg(func(ctx context.Context, req loginReq) (loginResp, error) {
	//		return loginResp{
	//			Token:   "token-for-" + req.Username,
	//			Profile: profileReq{UID: req.Username + "_uid"},
	//		}, nil
	//	}).Create()

	// Example 1: user service
	_ = register.Server("UserSample").
		Env("fea_user").
		RegName("Login", func(ctx context.Context, req loginReq) (loginResp, error) {
			return loginResp{
				Token:   "token-for-" + req.Username,
				Profile: profileReq{UID: req.Username + "_uid"},
			}, nil
		}).
		RegName("Profile", func(ctx context.Context, req profileReq) (profileResp, error) {
			return profileResp{
				UID:   req.UID,
				Email: fmt.Sprintf("%s@example.com", req.UID),
			}, nil
		}).
		Create()

	// Example 2: order service (custom Host)
	_ = register.Server("OrderSample").
		Env("fea_order").
		Version("v0.1.1").
		RegName("List", func(ctx context.Context, req orderListReq) (orderListResp, error) {
			items := []orderItem{
				{ID: 1, Title: "Book", Amount: 2},
				{ID: 2, Title: "Pen", Amount: 5},
			}
			return orderListResp{Items: items}, nil
		}).
		// map value 为 struct，覆盖 map[string] + 嵌套 struct schema
		RegName("MapOrders", func(ctx context.Context, req orderMapReq) (map[string]orderItem, error) {
			result := make(map[string]orderItem, len(req.IDs))
			for _, id := range req.IDs {
				result[fmt.Sprintf("order-%d", id)] = orderItem{ID: id, Title: "Item", Amount: 1}
			}
			return result, nil
		}).
		Create()

	// Example 3: // slice/map element
	_ = register.Server("MiscSample").
		Env("dev").
		RegName("EchoMap", func(ctx context.Context, req map[string]string) (map[string]string, error) {
			return req, nil
		}).
		RegName("EchoSlice", func(ctx context.Context, req []int) ([]int, error) {
			return req, nil
		}).
		Create()

	//Example 4: pointer参数/返回 + 自引用 struct
	_ = register.Server("AdvancedSample").
		Env("fea_advanced").
		// pointer in/out: BuildTypeSchema 会解引用
		RegName("PointerProfile", func(ctx context.Context, req *profileReq) (*profileResp, error) {
			if req == nil {
				return nil, fmt.Errorf("req is nil")
			}
			return &profileResp{
				UID:   req.UID,
				Email: fmt.Sprintf("%s@example.com", req.UID),
			}, nil
		}).
		// self-referencing struct: Child 指向自身类型
		RegName("Tree", func(ctx context.Context, req treeReq) (treeNode, error) {
			root := treeNode{ID: req.RootID}
			if len(req.ChildIDs) > 0 {
				root.Child = &treeNode{ID: req.ChildIDs[0]}
			}
			return root, nil
		}).
		Create()
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	Token   string     `json:"token"`
	Profile profileReq `json:"profile"`
}

type profileReq struct {
	UID string `json:"uid"`
}

type profileResp struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
}

type orderListReq struct {
	UID string `json:"uid"`
}

type orderMapReq struct {
	IDs []int `json:"ids"`
}

type orderItem struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Amount int    `json:"amount"`
}

type orderListResp struct {
	Items []orderItem `json:"items"`
}

type treeReq struct {
	RootID   int   `json:"root_id"`
	ChildIDs []int `json:"child_ids"`
}

type treeNode struct {
	ID    int       `json:"id"`
	Child *treeNode `json:"child,omitempty"`
	Name  string
}
