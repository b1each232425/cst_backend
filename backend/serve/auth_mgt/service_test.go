package auth_mgt

import (
	"context"
	"testing"

	"w2w.io/cmn"
	"w2w.io/null"
)

// equalAuthority 自定义比较函数，忽略Filter字段
func equalAuthority(a1, a2 *Authority) bool {
	if a1 == nil && a2 == nil {
		return true
	}
	if a1 == nil || a2 == nil {
		return false
	}

	// 比较Role字段（逐个比较TDomain的字段）
	if !equalTDomain(a1.Role, a2.Role) {
		return false
	}

	// 比较Domain字段
	if !equalTDomain(a1.Domain, a2.Domain) {
		return false
	}

	// 比较APIs数组长度
	if len(a1.APIs) != len(a2.APIs) {
		return false
	}

	// 逐个比较API，忽略Filter字段
	for i := range a1.APIs {
		if !equalTVDomainAPI(a1.APIs[i], a2.APIs[i]) {
			return false
		}
	}

	// 比较ReadableDomains字段（不关注元素顺序）
	if len(a1.AccessibleDomains) != len(a2.AccessibleDomains) {
		return false
	}
	// 将a2的ReadableDomains转换为map以便快速查找
	a2DomainsMap := make(map[int64]bool)
	for _, domain := range a2.AccessibleDomains {
		a2DomainsMap[domain] = true
	}
	// 检查a1中的每个域是否都存在于a2中
	for _, domain := range a1.AccessibleDomains {
		if !a2DomainsMap[domain] {
			return false
		}
	}

	return true
}

// equalTDomain 比较两个TDomain结构体，忽略Filter字段
func equalTDomain(d1, d2 cmn.TDomain) bool {
	return d1.ID == d2.ID &&
		d1.Name == d2.Name &&
		d1.Domain == d2.Domain &&
		d1.Priority == d2.Priority &&
		d1.DomainID == d2.DomainID &&
		d1.UpdatedBy == d2.UpdatedBy &&
		d1.UpdateTime == d2.UpdateTime &&
		d1.Creator == d2.Creator &&
		d1.CreateTime == d2.CreateTime &&
		d1.Remark == d2.Remark &&
		d1.Status == d2.Status
	// 忽略Addi和Filter字段
}

// equalTVDomainAPI 比较两个TVDomainAPI结构体，忽略Filter字段
func equalTVDomainAPI(api1, api2 cmn.TVDomainAPI) bool {
	return api1.ID == api2.ID &&
		api1.AuthDomainID == api2.AuthDomainID &&
		api1.DomainName == api2.DomainName &&
		api1.Domain == api2.Domain &&
		api1.Priority == api2.Priority &&
		api1.APIID == api2.APIID &&
		api1.APIName == api2.APIName &&
		api1.ExposePath == api2.ExposePath &&
		api1.AccessControlLevel == api2.AccessControlLevel &&
		api1.DomainID == api2.DomainID &&
		api1.GrantSource == api2.GrantSource &&
		api1.DataAccessMode == api2.DataAccessMode &&
		api1.CreateTime == api2.CreateTime &&
		api1.Remark == api2.Remark &&
		api1.Creator == api2.Creator &&
		api1.Status == api2.Status
	// 忽略DataScope、Addi和Filter字段
}

func TestGetUserAuthority(t *testing.T) {
	type args struct {
		sysUser    cmn.TUser
		forceError string
	}
	tests := []struct {
		name    string
		args    args
		wantA   *Authority
		wantErr bool
	}{
		{
			name: "正确查询数据库中的用户权限｜普通用户优先级角色",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003),
				},
				forceError: "",
			},
			wantA: &Authority{
				Role: cmn.TDomain{
					ID:       null.NewInt(2003, true),
					Name:     "教师",
					Domain:   "cst.school^teacher",
					Priority: null.IntFrom(7),
				},
				Domain: cmn.TDomain{
					ID:       null.NewInt(1999, true),
					Name:     "3min",
					Domain:   "cst.school",
					Priority: null.IntFrom(0),
				},
				APIs: []cmn.TVDomainAPI{
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20000),
						APIName:            null.StringFrom("题库管理模块"),
						ExposePath:         null.StringFrom("/teacher/question-bank"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20003),
						APIName:            null.StringFrom("试卷管理模块"),
						ExposePath:         null.StringFrom("/teacher/paper"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20004),
						APIName:            null.StringFrom("练习管理模块"),
						ExposePath:         null.StringFrom("/teacher/practice"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20005),
						APIName:            null.StringFrom("考试管理模块"),
						ExposePath:         null.StringFrom("/teacher/exam"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20006),
						APIName:            null.StringFrom("成绩管理模块"),
						ExposePath:         null.StringFrom("/teacher/grade"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20007),
						APIName:            null.StringFrom("学生管理模块"),
						ExposePath:         null.StringFrom("/teacher/student-management"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
				},
				AccessibleDomains: []int64{1999, 1998},
			},
			wantErr: false,
		},
		{
			name: "正确查询数据库中的用户权限｜超级管理员优先级角色",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2000),
				},
				forceError: "",
			},
			wantA: &Authority{
				Role: cmn.TDomain{
					ID:       null.NewInt(2000, true),
					Name:     "超级管理员",
					Domain:   "cst.school^superAdmin",
					Priority: null.IntFrom(0),
				},
				Domain: cmn.TDomain{
					ID:       null.NewInt(1999, true),
					Name:     "3min",
					Domain:   "cst.school",
					Priority: null.IntFrom(0),
				},
				APIs: []cmn.TVDomainAPI{
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20000),
						APIName:            null.StringFrom("题库管理模块"),
						ExposePath:         null.StringFrom("/teacher/question-bank"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20003),
						APIName:            null.StringFrom("试卷管理模块"),
						ExposePath:         null.StringFrom("/teacher/paper"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20004),
						APIName:            null.StringFrom("练习管理模块"),
						ExposePath:         null.StringFrom("/teacher/practice"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20005),
						APIName:            null.StringFrom("考试管理模块"),
						ExposePath:         null.StringFrom("/teacher/exam"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20006),
						APIName:            null.StringFrom("成绩管理模块"),
						ExposePath:         null.StringFrom("/teacher/grade"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20007),
						APIName:            null.StringFrom("学生管理模块"),
						ExposePath:         null.StringFrom("/teacher/student-management"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2000),
						DomainName:         null.StringFrom("超级管理员"),
						Domain:             null.StringFrom("cst.school^superAdmin"),
						Priority:           null.IntFrom(0),
						APIID:              null.IntFrom(20008),
						APIName:            null.StringFrom("用户管理模块"),
						ExposePath:         null.StringFrom("/teacher/user-management"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
				},
				AccessibleDomains: []int64{1998, 1999, 2009, 2011, 2013},
			},
			wantErr: false,
		},
		{
			name: "正确查询数据库中的用户权限｜普通管理员优先级角色",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1003),
					Role: null.IntFrom(2006),
				},
				forceError: "",
			},
			wantA: &Authority{
				Role: cmn.TDomain{
					ID:       null.NewInt(2006, true),
					Name:     "考点负责人",
					Domain:   "cst.school.examSite^admin",
					Priority: null.IntFrom(3),
				},
				Domain: cmn.TDomain{
					ID:       null.NewInt(2009, true),
					Name:     "考点",
					Domain:   "cst.school.examSite",
					Priority: null.IntFrom(0),
				},
				APIs:              []cmn.TVDomainAPI{},
				AccessibleDomains: []int64{1998, 1999, 2009, 2013},
			},
			wantErr: false,
		},
		{
			name: "sysUser为空",
			args: args{
				sysUser:    cmn.TUser{},
				forceError: "",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "用户的角色字段为域",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(1999), // 1999是域ID，不是角色ID
				},
				forceError: "",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发查询用户角色信息错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003), // 1999是域ID，不是角色ID
				},
				forceError: "QueryRole",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发查询用户域信息错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003), // 1999是域ID，不是角色ID
				},
				forceError: "QueryDomain",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发查询用户APIs信息错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003), // 1999是域ID，不是角色ID
				},
				forceError: "QueryAPIs",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发扫描用户APIs数据错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003), // 1999是域ID，不是角色ID
				},
				forceError: "ScanAPI",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发RowErr错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2003), // 1999是域ID，不是角色ID
				},
				forceError: "RowsErr",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "角色优先级无效",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2010), // 无角色优先级
				},
				forceError: "",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "非系统有效优先级角色",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1001),
					Role: null.IntFrom(2012), // 无角色优先级
				},
				forceError: "",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发CDomainPrioritySuperAdmin.Query错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2000),
				},
				forceError: "CDomainPrioritySuperAdmin.Query",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发CDomainPrioritySuperAdmin.Scan错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2000),
				},
				forceError: "CDomainPrioritySuperAdmin.Scan",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发CDomainPrioritySuperAdmin.RowsErr错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2000),
				},
				forceError: "CDomainPrioritySuperAdmin.RowsErr",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getParentDomains.QueryRow错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2003),
				},
				forceError: "getParentDomains.QueryRow",
			},
			wantA: &Authority{
				Role: cmn.TDomain{
					ID:       null.NewInt(2003, true),
					Name:     "教师",
					Domain:   "cst.school^teacher",
					Priority: null.IntFrom(7),
				},
				Domain: cmn.TDomain{
					ID:       null.NewInt(1999, true),
					Name:     "3min",
					Domain:   "cst.school",
					Priority: null.IntFrom(0),
				},
				APIs: []cmn.TVDomainAPI{
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20000),
						APIName:            null.StringFrom("题库管理模块"),
						ExposePath:         null.StringFrom("/teacher/question-bank"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20003),
						APIName:            null.StringFrom("试卷管理模块"),
						ExposePath:         null.StringFrom("/teacher/paper"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20004),
						APIName:            null.StringFrom("练习管理模块"),
						ExposePath:         null.StringFrom("/teacher/practice"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20005),
						APIName:            null.StringFrom("考试管理模块"),
						ExposePath:         null.StringFrom("/teacher/exam"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20006),
						APIName:            null.StringFrom("成绩管理模块"),
						ExposePath:         null.StringFrom("/teacher/grade"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
					{
						AuthDomainID:       null.IntFrom(2003),
						DomainName:         null.StringFrom("教师"),
						Domain:             null.StringFrom("cst.school^teacher"),
						Priority:           null.IntFrom(7),
						APIID:              null.IntFrom(20007),
						APIName:            null.StringFrom("学生管理模块"),
						ExposePath:         null.StringFrom("/teacher/student-management"),
						AccessControlLevel: null.StringFrom("4"),
						DataAccessMode:     null.StringFrom("full"),
						GrantSource:        null.StringFrom("api"),
					},
				},
				AccessibleDomains: []int64{1999},
			},
			wantErr: false,
		},
		{
			name: "触发getParentDomains.pgConn.nil错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getParentDomains.pgConn.nil",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getParentDomains.domainPath.empty错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2003),
				},
				forceError: "getParentDomains.domainPath.empty",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getChildDomains.pgConn.nil错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getChildDomains.pgConn.nil",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getChildDomains.domainPath.empty错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getChildDomains.domainPath.empty",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getChildDomains.Query错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getChildDomains.Query",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getChildDomains.Scan错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getChildDomains.Scan",
			},
			wantA:   nil,
			wantErr: true,
		},
		{
			name: "触发getChildDomains.RowsErr错误",
			args: args{
				sysUser: cmn.TUser{
					ID:   null.IntFrom(1002),
					Role: null.IntFrom(2006),
				},
				forceError: "getChildDomains.RowsErr",
			},
			wantA:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createMockContext(tt.args.sysUser, tt.args.forceError)

			gotA, err := GetUserAuthority(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserAuthority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 使用自定义比较函数，忽略Filter字段（解决reflect.DeepEqual因Filter字段导致的误报问题）
			t.Logf("🔍 使用自定义比较函数进行权限验证，忽略Filter等非业务字段")
			if !equalAuthority(gotA, tt.wantA) {
				t.Errorf("❌ GetUserAuthority() 权限验证失败：实际结果与期望结果不匹配")

				// 详细比对角色信息
				if gotA != nil && tt.wantA != nil {
					// 角色信息详细比对
					t.Logf("=== 角色信息比对 ===")
					if gotA.Role.ID != tt.wantA.Role.ID {
						t.Errorf("❌ 角色ID不匹配: 实际=%v, 期望=%v", gotA.Role.ID, tt.wantA.Role.ID)
					} else {
						t.Logf("✅ 角色ID匹配: %v", gotA.Role.ID)
					}

					if gotA.Role.Name != tt.wantA.Role.Name {
						t.Errorf("❌ 角色名称不匹配: 实际=%v, 期望=%v", gotA.Role.Name, tt.wantA.Role.Name)
					} else {
						t.Logf("✅ 角色名称匹配: %v", gotA.Role.Name)
					}

					if gotA.Role.Domain != tt.wantA.Role.Domain {
						t.Errorf("❌ 角色域不匹配: 实际=%v, 期望=%v", gotA.Role.Domain, tt.wantA.Role.Domain)
					} else {
						t.Logf("✅ 角色域匹配: %v", gotA.Role.Domain)
					}

					if gotA.Role.Priority != tt.wantA.Role.Priority {
						t.Errorf("❌ 角色优先级不匹配: 实际=%v, 期望=%v", gotA.Role.Priority, tt.wantA.Role.Priority)
					} else {
						t.Logf("✅ 角色优先级匹配: %v", gotA.Role.Priority)
					}

					// 域信息详细比对
					t.Logf("\n=== 域信息比对 ===")
					if gotA.Domain.ID != tt.wantA.Domain.ID {
						t.Errorf("❌ 域ID不匹配: 实际=%v, 期望=%v", gotA.Domain.ID, tt.wantA.Domain.ID)
					} else {
						t.Logf("✅ 域ID匹配: %v", gotA.Domain.ID)
					}

					if gotA.Domain.Name != tt.wantA.Domain.Name {
						t.Errorf("❌ 域名称不匹配: 实际=%v, 期望=%v", gotA.Domain.Name, tt.wantA.Domain.Name)
					} else {
						t.Logf("✅ 域名称匹配: %v", gotA.Domain.Name)
					}

					if gotA.Domain.Domain != tt.wantA.Domain.Domain {
						t.Errorf("❌ 域字符串不匹配: 实际=%v, 期望=%v", gotA.Domain.Domain, tt.wantA.Domain.Domain)
					} else {
						t.Logf("✅ 域字符串匹配: %v", gotA.Domain.Domain)
					}

					// API列表详细比对
					t.Logf("\n=== API列表比对 ===")
					if len(gotA.APIs) != len(tt.wantA.APIs) {
						t.Errorf("❌ API数量不匹配: 实际=%d, 期望=%d", len(gotA.APIs), len(tt.wantA.APIs))
					} else {
						t.Logf("✅ API数量匹配: %d", len(gotA.APIs))

						// 逐个比对API
						for i, gotAPI := range gotA.APIs {
							wantAPI := tt.wantA.APIs[i]
							t.Logf("\n--- API[%d] 详细比对 ---", i)

							if gotAPI.AuthDomainID != wantAPI.AuthDomainID {
								t.Errorf("❌ API[%d] AuthDomainID不匹配: 实际=%v, 期望=%v", i, gotAPI.AuthDomainID, wantAPI.AuthDomainID)
							} else {
								t.Logf("✅ API[%d] AuthDomainID匹配: %v", i, gotAPI.AuthDomainID)
							}

							if gotAPI.APIID != wantAPI.APIID {
								t.Errorf("❌ API[%d] APIID不匹配: 实际=%v, 期望=%v", i, gotAPI.APIID, wantAPI.APIID)
							} else {
								t.Logf("✅ API[%d] APIID匹配: %v", i, gotAPI.APIID)
							}

							if gotAPI.APIName != wantAPI.APIName {
								t.Errorf("❌ API[%d] APIName不匹配: 实际=%v, 期望=%v", i, gotAPI.APIName, wantAPI.APIName)
							} else {
								t.Logf("✅ API[%d] APIName匹配: %v", i, gotAPI.APIName)
							}

							if gotAPI.ExposePath != wantAPI.ExposePath {
								t.Errorf("❌ API[%d] ExposePath不匹配: 实际=%v, 期望=%v", i, gotAPI.ExposePath, wantAPI.ExposePath)
							} else {
								t.Logf("✅ API[%d] ExposePath匹配: %v", i, gotAPI.ExposePath)
							}

							if gotAPI.DataAccessMode != wantAPI.DataAccessMode {
								t.Errorf("❌ API[%d] DataAccessMode不匹配: 实际=%v, 期望=%v", i, gotAPI.DataAccessMode, wantAPI.DataAccessMode)
							} else {
								t.Logf("✅ API[%d] DataAccessMode匹配: %v", i, gotAPI.DataAccessMode)
							}

							if gotAPI.GrantSource != wantAPI.GrantSource {
								t.Errorf("❌ API[%d] GrantSource不匹配: 实际=%v, 期望=%v", i, gotAPI.GrantSource, wantAPI.GrantSource)
							} else {
								t.Logf("✅ API[%d] GrantSource匹配: %v", i, gotAPI.GrantSource)
							}

							// 显示API的完整信息
							t.Logf("API[%d] 完整信息: ID=%v, Name=%v, Path=%v, Mode=%v",
								i, gotAPI.APIID, gotAPI.APIName, gotAPI.ExposePath, gotAPI.DataAccessMode)
						}
					}

					// ReadableDomains详细比对
					t.Logf("\n=== 可读域列表比对 ===")
					if len(gotA.AccessibleDomains) != len(tt.wantA.AccessibleDomains) {
						t.Errorf("❌ 可读域数量不匹配: 实际=%d, 期望=%d", len(gotA.AccessibleDomains), len(tt.wantA.AccessibleDomains))
					} else {
						t.Logf("✅ 可读域数量匹配: %d", len(gotA.AccessibleDomains))

						// 逐个比对可读域ID
						for i, gotDomainID := range gotA.AccessibleDomains {
							wantDomainID := tt.wantA.AccessibleDomains[i]
							if gotDomainID != wantDomainID {
								t.Errorf("❌ 可读域[%d] ID不匹配: 实际=%d, 期望=%d", i, gotDomainID, wantDomainID)
							} else {
								t.Logf("✅ 可读域[%d] ID匹配: %d", i, gotDomainID)
							}
						}
					}
					t.Logf("实际可读域列表: %v", gotA.AccessibleDomains)
					t.Logf("期望可读域列表: %v", tt.wantA.AccessibleDomains)

					// 测试结果汇总
					t.Logf("\n=== 测试结果汇总 ===")
					t.Logf("测试用例: %s", tt.name)
					t.Logf("角色ID: %v", tt.args.sysUser.ID)
					t.Logf("期望角色: %v", tt.wantA.Role.Name)
					t.Logf("期望域: %v", tt.wantA.Domain.Name)
					t.Logf("期望API数量: %d", len(tt.wantA.APIs))
					t.Logf("实际API数量: %d", len(gotA.APIs))
					t.Logf("期望可读域数量: %d", len(tt.wantA.AccessibleDomains))
					t.Logf("实际可读域数量: %d", len(gotA.AccessibleDomains))
					t.Logf("===================")

				} else {
					t.Errorf("❌ 权限对象为空: 实际=%v, 期望=%v", gotA, tt.wantA)
				}
			} else {
				t.Logf("✅ 权限验证通过：实际结果与期望结果完全匹配")
				t.Logf("✅ 测试通过: %s", tt.name)
				if gotA != nil {
					t.Logf("角色: %v (ID: %v)", gotA.Role.Name, gotA.Role.ID)
					t.Logf("域: %v (ID: %v)", gotA.Domain.Name, gotA.Domain.ID)
					t.Logf("API权限数量: %d", len(gotA.APIs))
					for i, api := range gotA.APIs {
						t.Logf("  API[%d]: %v -> %v", i, api.APIName, api.ExposePath)
					}
					t.Logf("可读域数量: %d", len(gotA.AccessibleDomains))
					t.Logf("可读域列表: %v", gotA.AccessibleDomains)
				}
			}
		})
	}
}

func TestCheckUserAPIAccessible(t *testing.T) {
	type args struct {
		ctx        context.Context
		authority  *Authority
		apiPath    string
		accessMode string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "full权限｜API可读",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionQuery,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "full权限｜API可写",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionAdd,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "full权限｜API可完全访问",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionFull,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "query权限｜API可查询数据",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("query"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionQuery,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "query权限｜API不可添加数据",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("query"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionAdd,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "query权限｜API不可完全访问",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("query"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionFull,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "add权限｜API不可查询数据",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("add"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionQuery,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "edit权限｜API可编辑",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("edit"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionEdit,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "delete权限｜API可删除数据",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							AccessAction:       null.StringFrom("delete"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionDelete,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "非合法访问类型",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("read"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: "invalid_mode",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "authority为空",
			args: args{
				ctx:        createMockContext(cmn.TUser{ID: null.IntFrom(1001)}, ""),
				authority:  nil,
				apiPath:    "/teacher/question-bank",
				accessMode: "invalid_mode",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "checkUserAPIAccessible.authority.nil错误",
			args: args{
				ctx: createMockContext(cmn.TUser{}, "checkUserAPIAccessible.authority.nil"),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				apiPath:    "/teacher/question-bank",
				accessMode: CAPIAccessActionFull,
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckUserAPIAccessible(tt.args.ctx, tt.args.authority, tt.args.apiPath, tt.args.accessMode)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckUserAPIAccessible() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckUserAPIAccessible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDomainRelationship(t *testing.T) {
	type args struct {
		ctx          context.Context
		authority    *Authority
		targetDomain string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "本域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school",
			},
			want:    CDomainRelationshipSelf,
			wantErr: false,
		},
		{
			name: "相邻父域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst",
			},
			want:    CDomainRelationshipParent,
			wantErr: false,
		},
		{
			name: "非相邻父域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school.class^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school.class",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst",
			},
			want:    CDomainRelationshipParent,
			wantErr: false,
		},
		{
			name: "相邻子域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school.class",
			},
			want:    CDomainRelationshipChild,
			wantErr: false,
		},
		{
			name: "非相邻子域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school.class.room",
			},
			want:    CDomainRelationshipChild,
			wantErr: false,
		},
		{
			name: "同级平行域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school2",
			},
			want:    CDomainRelationshipPeer,
			wantErr: false,
		},
		{
			name: "非同级平行域",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school2.class",
			},
			want:    CDomainRelationshipPeer,
			wantErr: false,
		},
		{
			name: "目标域非格式不合法1",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school2.class^member",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "目标域非格式不合法2",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school2*class",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "目标域非格式不合法3",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "authority为空",
			args: args{
				ctx:          context.Background(),
				authority:    nil,
				targetDomain: "cst.school",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "用户的域为空",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "GetDomainRelationship.isParentOf.empty错误",
			args: args{
				ctx: createMockContext(cmn.TUser{}, "GetDomainRelationship.isParentOf.empty"),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school.class",
			},
			want:    CDomainRelationshipPeer,
			wantErr: false,
		},
		{
			name: "用户的域格式为角色",
			args: args{
				ctx: context.Background(),
				authority: &Authority{
					Role: cmn.TDomain{
						ID:       null.NewInt(2003, true),
						Name:     "教师",
						Domain:   "cst.school^teacher",
						Priority: null.IntFrom(7),
					},
					Domain: cmn.TDomain{
						ID:       null.NewInt(1999, true),
						Name:     "3min",
						Domain:   "cst.school^user",
						Priority: null.IntFrom(0),
					},
					APIs: []cmn.TVDomainAPI{
						{
							AuthDomainID:       null.IntFrom(2003),
							DomainName:         null.StringFrom("教师"),
							Domain:             null.StringFrom("cst.school^teacher"),
							Priority:           null.IntFrom(7),
							APIID:              null.IntFrom(20000),
							APIName:            null.StringFrom("题库管理模块"),
							ExposePath:         null.StringFrom("/teacher/question-bank"),
							AccessControlLevel: null.StringFrom("4"),
							DataAccessMode:     null.StringFrom("full"),
							GrantSource:        null.StringFrom("api"),
						},
					},
					AccessibleDomains: []int64{1999, 1998},
				},
				targetDomain: "cst.school",
			},
			want:    CDomainRelationshipSelf,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDomainRelationship(tt.args.ctx, tt.args.authority, tt.args.targetDomain)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDomainRelationship() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDomainRelationship() = %v, want %v", got, tt.want)
			}
		})
	}
}
