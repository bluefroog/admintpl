package domain

// Ztree 通用树形结构
// 说明：返回给前端的为扁平列表，由前端 zTree 插件根据 pId 自行组装成树
type Ztree struct {
	ID       uint   `json:"id"`      // 节点ID
	ParentID uint   `json:"pId"`     // 节点父ID
	Name     string `json:"name"`    // 节点名称
	Title    string `json:"title"`   // 节点标题
	Checked  bool   `json:"checked"` // 是否勾选
	Open     bool   `json:"open"`    // 是否展开
	Nocheck  bool   `json:"nocheck"` // 是否允许勾选
}
