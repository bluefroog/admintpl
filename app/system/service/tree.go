package service

import "github.com/bluefroog/admintpl/app/system/domain"

// deptListToZtree 部门实体列表转换为树节点列表
// @author: [bluefrog](https://github.com/freewu)
// @function: deptListToZtree
// @description: 部门列表转树节点列表，checkedIds 非空时标记勾选
// @param: list []domain.Department
// @param: checkedIds []uint
// @return: []domain.Ztree
func deptListToZtree(list []domain.Department, checkedIds []uint) []domain.Ztree {
	isCheck := len(checkedIds) > 0
	checked := make(map[uint]bool, len(checkedIds))
	for _, id := range checkedIds {
		checked[id] = true
	}
	result := make([]domain.Ztree, 0, len(list))
	for i := range list {
		node := domain.Ztree{
			ID:       list[i].ID,
			ParentID: list[i].ParentID,
			Name:     list[i].Name,
			Title:    list[i].Name,
		}
		if isCheck {
			node.Checked = checked[list[i].ID]
		}
		result = append(result, node)
	}
	return result
}

// menuListToZtree 菜单实体列表转换为树节点列表
// @author: [bluefrog](https://github.com/freewu)
// @function: menuListToZtree
// @description: 菜单列表转树节点列表，checkedIds 非空时标记勾选
// @param: list []domain.Menu
// @param: checkedIds []uint
// @return: []domain.Ztree
func menuListToZtree(list []domain.Menu, checkedIds []uint) []domain.Ztree {
	isCheck := len(checkedIds) > 0
	checked := make(map[uint]bool, len(checkedIds))
	for _, id := range checkedIds {
		checked[id] = true
	}
	result := make([]domain.Ztree, 0, len(list))
	for i := range list {
		node := domain.Ztree{
			ID:       list[i].ID,
			ParentID: list[i].ParentID,
			Name:     list[i].Name,
			Title:    list[i].Name,
		}
		if isCheck {
			node.Checked = checked[list[i].ID]
		}
		result = append(result, node)
	}
	return result
}
