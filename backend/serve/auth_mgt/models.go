package auth_mgt

import "w2w.io/cmn"

const (
	CDataAccessModeFull  = "full"  // API完全访问
	CDataAccessModeRead  = "read"  // API只读访问
	CDataAccessModeWrite = "write" // API只写访问

	CDomainRelationshipSelf   = "self"    // 域关系：自身域
	CDomainRelationshipParent = "parent"  // 域关系：父域
	CDomainRelationshipChild  = "child"   // 域关系：子域
	CDomainRelationshipPeer   = "sibling" // 域关系：平行域

	CDomainPrioritySuperAdmin int64 = 0 // 域优先级：超级管理员
	CDomainPriorityAdmin      int64 = 3 // 域优先级：普通管理员
	CDomainPriorityUser       int64 = 7 // 域优先级：普通用户
)

type Authority struct {
	Role            cmn.TDomain       // 用户角色
	APIs            []cmn.TVDomainAPI // 用户可访问的API列表
	Domain          cmn.TDomain       // 用户所在域
	ReadableDomains []int64           // 用户可读域列表
}
