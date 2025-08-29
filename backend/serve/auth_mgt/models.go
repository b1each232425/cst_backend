package auth_mgt

import "w2w.io/cmn"

const (
	CAPIAccessActionFull   = "full"   // API访问操作：完全访问
	CAPIAccessActionCreate = "create" // API访问操作：添加数据
	CAPIAccessActionRead   = "read"   // API访问操作：查询数据
	CAPIAccessActionUpdate = "update" // API访问操作：更新数据
	CAPIAccessActionDelete = "delete" // API访问操作：删除数据

	CDomainRelationshipSelf   = "self"    // 域关系：自身域
	CDomainRelationshipParent = "parent"  // 域关系：父域
	CDomainRelationshipChild  = "child"   // 域关系：子域
	CDomainRelationshipPeer   = "sibling" // 域关系：平行域

	CDomainPrioritySuperAdmin int64 = 0 // 域优先级：超级管理员
	CDomainPriorityAdmin      int64 = 3 // 域优先级：普通管理员
	CDomainPriorityUser       int64 = 7 // 域优先级：普通用户
)

// Authority 用户权限信息
type Authority struct {
	Role              cmn.TDomain       // 用户角色
	APIs              []cmn.TVDomainAPI // 用户的API列表
	Domain            cmn.TDomain       // 用户所在域
	AccessibleDomains []int64           // 用户可访问域列表
}

// DomainData 域数据
type DomainData struct {
	Base cmn.TDomain `json:"base"` // 域的基本信息
	APIs []*cmn.TAPI `json:"apis"` // 域的API列表
}
