package auth_mgt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
	"w2w.io/null"
)

// TestHandleQuerySelectableAPIs 测试查询可选API的处理函数
func TestHandleQuerySelectableAPIs(t *testing.T) {
	type args struct {
		method       string
		parentDomain string
		forceError   string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
		wantErr    bool
	}{
		{
			name: "成功查询机构级别API｜GET方法",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "",
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询部门级别API｜有效父域",
			args: args{
				method:       "GET",
				parentDomain: "assess",
				forceError:   "",
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "失败｜无效HTTP方法",
			args: args{
				method:       "POST",
				parentDomain: "",
				forceError:   "",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜无效父域格式",
			args: args{
				method:       "GET",
				parentDomain: "invalid^domain^format",
				forceError:   "",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜强制querySelectableAPIs错误",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "querySelectableAPIs",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜强制JSON序列化错误",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "json.Marshal",
			},
			wantStatus: -1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建HTTP请求
			req := httptest.NewRequest(tt.args.method, "/api/querySelectableAPIs", nil)
			if tt.args.parentDomain != "" {
				q := req.URL.Query()
				q.Set("parentDomain", tt.args.parentDomain)
				req.URL.RawQuery = q.Encode()
			}

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 创建上下文并设置必要的值
			ctx := context.Background()
			if tt.args.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.args.forceError)
			}

			// 创建cmn.ServiceCtx对象并设置到上下文中
			q := &cmn.ServiceCtx{
				R:   req,
				W:   w,
				Msg: &cmn.ReplyProto{},
			}
			ctx = context.WithValue(ctx, cmn.QNearKey, q)

			// 创建handler并调用方法
			h := NewHandler()
			h.HandleQuerySelectableAPIs(ctx)

			// 验证结果
			if tt.wantErr {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleQuerySelectableAPIs() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err == nil {
					t.Errorf("HandleQuerySelectableAPIs() expected error but got none")
				}
			} else {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleQuerySelectableAPIs() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err != nil {
					t.Errorf("HandleQuerySelectableAPIs() unexpected error = %v", q.Err)
				}

				// 验证返回的数据是否为有效的JSON
				if len(q.Msg.Data) > 0 {
					var apis []interface{}
					if err := json.Unmarshal(q.Msg.Data, &apis); err != nil {
						t.Errorf("HandleQuerySelectableAPIs() returned invalid JSON data: %v", err)
					}
				}
			}

			// 输出调试信息
			if testing.Verbose() {
				t.Logf("测试用例: %s", tt.name)
				t.Logf("  请求方法: %s", tt.args.method)
				t.Logf("  父域: %s", tt.args.parentDomain)
				t.Logf("  强制错误: %s", tt.args.forceError)
				t.Logf("  响应状态: %d", q.Msg.Status)
				t.Logf("  响应消息: %s", q.Msg.Msg)
				if q.Err != nil {
					t.Logf("  错误信息: %v", q.Err)
				}
			}
		})
	}
}

// cleanupTestData 清理测试数据
// 删除t_domain表中remark字段为'test'的数据及其关联的t_domain_api数据
func cleanupTestData(t *testing.T) {
	ctx := context.Background()
	conn := cmn.GetPgxConn()
	if conn == nil {
		t.Logf("获取数据库连接失败，跳过清理")
		return
	}

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始事务失败: %v", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	// 查询需要删除的域ID
	var domainIDs []int64
	rows, err := tx.Query(ctx, "SELECT id FROM t_domain WHERE remark = 'test'")
	if err != nil {
		t.Logf("查询测试域失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var domainID int64
		if err := rows.Scan(&domainID); err != nil {
			t.Logf("扫描域ID失败: %v", err)
			continue
		}
		domainIDs = append(domainIDs, domainID)
	}

	if len(domainIDs) == 0 {
		// 没有需要清理的数据，提交事务
		if err := tx.Commit(ctx); err != nil {
			t.Logf("提交事务失败: %v", err)
		}
		return
	}

	// 删除t_domain_api表中的关联数据
	_, err = tx.Exec(ctx, "DELETE FROM t_domain_api WHERE domain = ANY($1)", domainIDs)
	if err != nil {
		t.Logf("删除域API关联数据失败: %v", err)
		return
	}

	// 删除t_domain表中的测试数据
	_, err = tx.Exec(ctx, "DELETE FROM t_domain WHERE remark = 'test'")
	if err != nil {
		t.Logf("删除测试域失败: %v", err)
		return
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		t.Logf("提交事务失败: %v", err)
		return
	}

	t.Logf("成功清理了 %d 个测试域及其关联数据", len(domainIDs))
}

// TestHandleDomain 测试域管理的处理函数
func TestHandleDomain(t *testing.T) {
	type args struct {
		method     string
		body       interface{}
		forceError string
		userID     int64
		userRole   int64
		params     map[string]string // 查询参数
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
		wantErr    bool
	}{
		// 以下为通用测试用例
		{
			name: "失败｜用户未登录",
			args: args{
				method:     "POST",
				body:       nil,
				forceError: "",
				userID:     0,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜非超级管理员权限",
			args: args{
				method:     "POST",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜请求体为空",
			args: args{
				method:     "POST",
				body:       "",
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜不支持的请求方法",
			args: args{
				method:     "PATCH",
				body:       "",
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},

		// 以下为POST方法的测试用例
		{
			name: "成功创建机构｜POST方法｜有效域数据",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校",
						Domain: "testSchool",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   20000, // 超级管理员
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功创建部门｜POST方法｜有效域数据",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校.课程部",
						Domain: "testSchool.course",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   20000, // 超级管理员
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功创建角色｜POST方法｜有效域数据",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:     "测试学校.课程部.课程管理员",
						Priority: null.NewInt(CDomainPriorityAdmin, true),
						Domain:   "testSchool.course^admin",
						Remark:   null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   20000, // 超级管理员
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "创建机构失败｜POST方法｜英文代号不合法",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校",
						Domain: "testSchool-",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   20000, // 超级管理员
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "创建部门失败｜POST方法｜所属机构不存在",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校2.课程部",
						Domain: "testSchool2.course",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   20000, // 超级管理员
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "创建机构失败｜POST方法｜创建者非超级管理员角色",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校3",
						Domain: "testSchool3",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
					},
				},
				forceError: "",
				userID:     1,
				userRole:   2001, // 普通管理员
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜域英文代码为空",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校",
						Domain: "",
					},
					APIs: []*cmn.TAPI{},
				},
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜域名称为空",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "",
						Domain: "test.school",
					},
					APIs: []*cmn.TAPI{},
				},
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜无效域格式",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试",
						Domain: "invalid^domain^format",
					},
					APIs: []*cmn.TAPI{},
				},
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜角色优先级无效",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:     "测试角色",
						Domain:   "test.school^user",
						Priority: null.NewInt(999, true),
					},
					APIs: []*cmn.TAPI{},
				},
				forceError: "",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制GetUserAuthority错误",
			args: args{
				method:     "POST",
				body:       nil,
				forceError: "GetUserAuthority",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制io.ReadAll错误",
			args: args{
				method:     "POST",
				body:       "test",
				forceError: "io.ReadAll",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制JSON反序列化错误",
			args: args{
				method:     "POST",
				body:       DomainData{},
				forceError: "json.Unmarshal",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制JSON反序列化body.data错误",
			args: args{
				method:     "POST",
				body:       DomainData{},
				forceError: "json.UnmarshalDomainData",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制GetPgxConn错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校4",
						Domain: "testSchool4",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "GetPgxConn",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制tx.Begin错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校4",
						Domain: "testSchool4",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "tx.Begin",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制tx.QueryRow错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校4",
						Domain: "testSchool4",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "tx.QueryRow",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制tx.Exec错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校5",
						Domain: "testSchool5",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "tx.Exec",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜POST方法｜强制json.MarshalResponse错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校6",
						Domain: "testSchool6",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "json.MarshalResponse",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: -1,
			wantErr:    false,
		},
		{
			name: "失败｜POST方法｜强制io.Close错误",
			args: args{
				method: "POST",
				body: DomainData{
					Base: cmn.TDomain{
						Name:   "测试学校4",
						Domain: "testSchool4",
						Remark: null.StringFrom("test"),
					},
					APIs: []*cmn.TAPI{
						{
							ID: null.NewInt(20011, true),
						},
						{
							ID: null.NewInt(20012, true),
						},
						{
							ID: null.NewInt(20013, true),
						},
						{
							ID: null.NewInt(20014, true),
						},
						{
							ID: null.NewInt(20015, true),
						},
					},
				},
				forceError: "io.Close",
				userID:     1,
				userRole:   20000,
			},
			wantStatus: 0,
			wantErr:    false,
		},

		// 以下为GET方法的测试用例
		{
			name: "成功查询域列表｜GET方法｜无筛选条件",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带状态筛选",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"status": "01"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带域筛选",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"domain": "testSchool"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带模糊查询",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"fuzzyCondition": "测试"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带分页参数",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "5"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带分页参数｜页大小超过1000",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10000"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜带父域参数",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10", "parentDomain": "testSchool"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询域列表｜GET方法｜只查询角色",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10", "onlyRole": "true"},
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "失败｜GET方法｜无效域格式",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"domain": "invalid^domain^format"},
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜无效父域格式",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"parentDomain": "invalid^parent^format"},
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制GetPgxConn错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "GetPgxConn",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制QueryDomains错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "QueryDomains",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制ScanDomains错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "ScanDomains",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制QueryDomainAPIs错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "QueryDomainAPIs",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制ScanDomainAPIs错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "ScanDomainAPIs",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制json.MarshalDomainList错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "json.MarshalDomainList",
				userID:     1,
				userRole:   20000,
				params:     nil,
			},
			wantStatus: -1,
			wantErr:    false,
		},
		{
			name: "失败｜GET方法｜强制apiRows.Err错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "apiRows.Err",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10", "onlyRole": "true"},
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制rows.Err错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "rows.Err",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10", "onlyRole": "true"},
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜GET方法｜强制QueryDomainCount错误",
			args: args{
				method:     "GET",
				body:       nil,
				forceError: "QueryDomainCount",
				userID:     1,
				userRole:   20000,
				params:     map[string]string{"page": "1", "pageSize": "10", "onlyRole": "true"},
			},
			wantStatus: -1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构造请求体
			var reqBody *strings.Reader
			if tt.args.body != nil {
				if bodyStr, ok := tt.args.body.(string); ok {
					// 如果是字符串，直接使用
					reqBody = strings.NewReader(bodyStr)
				} else {
					// 如果是结构体，先序列化为JSON
					domainDataBytes, _ := json.Marshal(tt.args.body)
					reqProto := cmn.ReqProto{
						Data: domainDataBytes,
					}
					reqBytes, _ := json.Marshal(reqProto)
					reqBody = strings.NewReader(string(reqBytes))
				}
			} else {
				reqBody = strings.NewReader("")
			}

			// 创建HTTP请求
			reqURL := "/api/domain"
			if tt.args.params != nil && len(tt.args.params) > 0 {
				var queryParams []string
				for key, value := range tt.args.params {
					queryParams = append(queryParams, key+"="+value)
				}
				reqURL += "?" + strings.Join(queryParams, "&")
			}
			req := httptest.NewRequest(tt.args.method, reqURL, reqBody)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 创建上下文并设置必要的值
			ctx := context.Background()
			if tt.args.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.args.forceError)
			}

			// 创建cmn.ServiceCtx对象并设置到上下文中
			q := &cmn.ServiceCtx{
				R:   req,
				W:   w,
				Msg: &cmn.ReplyProto{},
			}

			// 设置用户信息
			if tt.args.userID > 0 {
				q.SysUser = &cmn.TUser{
					ID:   null.NewInt(tt.args.userID, true),
					Role: null.NewInt(tt.args.userRole, true),
				}
			}

			ctx = context.WithValue(ctx, cmn.QNearKey, q)

			// 创建handler并调用方法
			h := NewHandler()
			h.HandleDomain(ctx)

			// 验证结果
			if tt.wantErr {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleDomain() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err == nil {
					t.Errorf("HandleDomain() expected error but got none")
				}
			} else {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleDomain() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err != nil {
					t.Errorf("HandleDomain() unexpected error = %v", q.Err)
				}

				// 验证返回的数据是否为有效的JSON
				if len(q.Msg.Data) > 0 {
					if tt.args.method == "GET" {
						// GET方法返回域列表
						var domainList []DomainData
						if err := json.Unmarshal(q.Msg.Data, &domainList); err != nil {
							t.Errorf("HandleDomain() GET returned invalid JSON data: %v", err)
						}
					} else {
						// POST方法返回单个域数据
						var domainData DomainData
						if err := json.Unmarshal(q.Msg.Data, &domainData); err != nil {
							t.Errorf("HandleDomain() POST returned invalid JSON data: %v", err)
						}
					}
				}
			}

			// 输出调试信息
			if testing.Verbose() {
				t.Logf("测试用例: %s", tt.name)
				t.Logf("  请求方法: %s", tt.args.method)
				t.Logf("  用户ID: %d", tt.args.userID)
				t.Logf("  用户角色: %d", tt.args.userRole)
				t.Logf("  强制错误: %s", tt.args.forceError)
				t.Logf("  响应状态: %d", q.Msg.Status)
				t.Logf("  响应消息: %s", q.Msg.Msg)
				if q.Err != nil {
					t.Logf("  错误信息: %v", q.Err)
				}
			}
		})
	}

	// 清理测试数据
	cleanupTestData(t)
}
