package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DataScope 数据权限服务
// @author: [bluefrog](https://github.com/freewu)
// @function: DataScope
// @description: 按登录用户的角色数据范围（sys_role.data_scope）为列表查询追加过滤条件，
//
//	语义与若依 DataScopeAspect（@DataScope 注解）保持一致：
//	1=全部数据权限 2=自定义数据权限 3=本部门数据权限 4=本部门及以下数据权限 5=仅本人数据权限
type DataScope struct{}

// 数据范围取值，与 sys_role.data_scope 保持一致
const (
	DataScopeAll          uint = 1 // 全部数据权限
	DataScopeCustom       uint = 2 // 自定义数据权限（按 sys_role_dept 中配置的部门过滤）
	DataScopeDepartment   uint = 3 // 本部门数据权限
	DataScopeDeptAndChild uint = 4 // 本部门及以下数据权限
	DataScopeSelf         uint = 5 // 仅本人数据权限
)

// DataScopeOptions 数据权限过滤选项（等价若依 @DataScope(deptAlias, userAlias)）
type DataScopeOptions struct {
	DeptColumn string // 部门ID列，如 "department_id"、"u.department_id"
	UserColumn string // 用户ID列，如 "id"；为空表示该查询不支持「仅本人」数据权限（该条件下恒不成立）
}

// roleScope 登录用户的角色及其数据范围
type roleScope struct {
	ID        uint // 角色ID（自定义数据权限时用于关联 sys_role_dept）
	DataScope uint // 数据范围
}

// Apply 按登录用户的数据权限为查询追加过滤条件
// @author: [bluefrog](https://github.com/freewu)
// @function: Apply
// @description: 无登录用户（认证关闭）、超级管理员或数据权限开关关闭时不追加任何条件
// @param: db *gorm.DB 待过滤的查询
// @param: loginUser *domain.LoginUser 登录用户（来自 middleware.GetLoginUser）
// @param: options DataScopeOptions 数据权限过滤选项
// @return: *gorm.DB
func (s DataScope) Apply(db *gorm.DB, loginUser *domain.LoginUser, options DataScopeOptions) *gorm.DB {
	conditions, args, limited := s.userScope(loginUser, options)
	return s.applyConditions(db, conditions, args, limited)
}

// applyConditions 把数据范围条件追加到查询上（拆出便于单元测试）
// @author: [bluefrog](https://github.com/freewu)
// @function: applyConditions
// @description: limited=false 时原样返回查询
// @param: db *gorm.DB 待过滤的查询
// @param: conditions string 过滤条件
// @param: args []interface{} 条件参数
// @param: limited bool 是否需要追加条件
// @return: *gorm.DB
func (s DataScope) applyConditions(db *gorm.DB, conditions string, args []interface{}, limited bool) *gorm.DB {
	if !limited {
		return db
	}
	return db.Where(conditions, args...)
}

// ApplyToRoles 角色列表的数据权限过滤
// @author: [bluefrog](https://github.com/freewu)
// @function: ApplyToRoles
// @description: 角色自身没有部门，若依通过 JOIN 用户/部门表实现过滤，这里用子查询达到同样效果
//
//	（避免列表查询出现 JOIN 与列名歧义，同时保证统计总数与分页结果一致）
//
// @param: db *gorm.DB 角色查询
// @param: loginUser *domain.LoginUser 登录用户
// @return: *gorm.DB
func (s DataScope) ApplyToRoles(db *gorm.DB, loginUser *domain.LoginUser) *gorm.DB {
	conditions, args, limited := s.userScope(loginUser, DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"})
	if !limited {
		return db
	}
	userIDs := database.DB.Model(&domain.User{}).Select("id").Where(conditions, args...)
	roleIDs := database.DB.Model(&domain.UserRole{}).Select("role_id").Where("user_id IN (?)", userIDs)
	return db.Where("role_id IN (?)", roleIDs)
}

// userScope 计算登录用户的数据范围过滤条件
// @author: [bluefrog](https://github.com/freewu)
// @function: userScope
// @description: 关闭数据权限的三种情形：未登录（认证关闭）、超级管理员、开关显式配置为 false
// @param: loginUser *domain.LoginUser 登录用户
// @param: options DataScopeOptions 数据权限过滤选项
// @return: conditions string 过滤条件, args []interface{} 参数, limited bool 是否需要追加条件
func (s DataScope) userScope(loginUser *domain.LoginUser, options DataScopeOptions) (conditions string, args []interface{}, limited bool) {
	// 数据库未初始化（如单元测试）时不追加条件
	if database.DB == nil {
		return "", nil, false
	}
	if loginUser == nil || loginUser.IsAdmin {
		return "", nil, false
	}
	// 数据权限开关：仅在显式配置为 false 时关闭（配置文件缺失该项时按开启处理）
	if config.IsSet("system.data-scope.enabled") && !config.Bool("system.data-scope.enabled") {
		return "", nil, false
	}
	return s.condition(s.roleScopes(loginUser.UserID), options, loginUser.DepartmentID, loginUser.UserID)
}

// roleScopes 查询登录用户的有效角色（启用且未删除）及其数据范围
// @author: [bluefrog](https://github.com/freewu)
// @function: roleScopes
// @description: 查询失败时返回空列表（不追加条件），仅记录错误日志，不阻断业务查询
// @param: userID uint 用户ID
// @return: []roleScope
func (s DataScope) roleScopes(userID uint) []roleScope {
	var roleIDs []uint
	if err := database.DB.Model(&domain.UserRole{}).
		Where("user_id = ?", userID).
		Distinct().
		Pluck("role_id", &roleIDs).Error; err != nil {
		log.Errorf("数据权限：查询用户 %d 的角色失败：%v", userID, err)
		return nil
	}
	if len(roleIDs) == 0 {
		return nil
	}
	var roles []domain.Role
	if err := database.DB.Model(&domain.Role{}).
		Where("role_id IN (?) AND status = 0 AND del_flag = 0", roleIDs).
		Find(&roles).Error; err != nil {
		log.Errorf("数据权限：查询角色数据范围失败：%v", err)
		return nil
	}
	scopes := make([]roleScope, 0, len(roles))
	for _, role := range roles {
		scopes = append(scopes, roleScope{ID: role.ID, DataScope: role.DataScope})
	}
	return scopes
}

// condition 组装数据范围过滤条件
// @author: [bluefrog](https://github.com/freewu)
// @function: condition
// @description: 多个角色的数据范围按 OR 拼接（任一角色为「全部数据权限」则不限制）；
//
//	自定义数据权限按角色分别生成条件，其余数据范围去重；无任何可用条件时不做限制（与若依一致）
//
// @param: scopes []roleScope 用户角色及数据范围
// @param: options DataScopeOptions 数据权限过滤选项
// @param: deptID uint 登录用户所属部门ID
// @param: userID uint 登录用户ID
// @return: conditions string, args []interface{}, limited bool
func (s DataScope) condition(scopes []roleScope, options DataScopeOptions, deptID, userID uint) (string, []interface{}, bool) {
	if len(scopes) == 0 { // 无角色（或未配置数据范围）：与若依一致，不追加条件
		return "", nil, false
	}
	conditions := make([]string, 0, len(scopes))
	args := make([]interface{}, 0, len(scopes)*3)
	seen := make(map[uint]bool, len(scopes)) // 已处理的数据范围（自定义数据权限不参与去重）
	unknown := make([]uint, 0)
	for _, scope := range scopes {
		if scope.DataScope == DataScopeAll { // 任一角色为全部数据权限：直接不限制
			return "", nil, false
		}
		if scope.DataScope != DataScopeCustom {
			if seen[scope.DataScope] {
				continue
			}
			seen[scope.DataScope] = true
		}
		switch scope.DataScope {
		case DataScopeCustom: // 自定义数据权限：按角色关联的部门过滤
			conditions = append(conditions, fmt.Sprintf(
				"%s IN (SELECT dept_id FROM %s WHERE role_id = ?)",
				options.DeptColumn, tableName(&domain.RoleDept{})))
			args = append(args, scope.ID)
		case DataScopeDepartment: // 本部门数据权限
			conditions = append(conditions, fmt.Sprintf("%s = ?", options.DeptColumn))
			args = append(args, deptID)
		case DataScopeDeptAndChild: // 本部门及以下数据权限
			conditions = append(conditions, fmt.Sprintf(
				"%s IN (SELECT dept_id FROM %s WHERE dept_id = ? OR FIND_IN_SET(?, ancestors))",
				options.DeptColumn, tableName(&domain.Department{})))
			args = append(args, deptID, deptID)
		case DataScopeSelf: // 仅本人数据权限
			if options.UserColumn == "" { // 该查询没有用户ID列时无法表达「仅本人」，恒不成立
				conditions = append(conditions, "1 = 0")
				break
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", options.UserColumn))
			args = append(args, userID)
		default: // 无法识别的数据范围：跳过（与若依一致），仅告警便于排查
			unknown = append(unknown, scope.ID)
		}
	}
	if len(unknown) > 0 {
		log.Warnf("数据权限：角色 %v 的数据范围无法识别（有效值 1~5），已跳过该角色", unknown)
	}
	if len(conditions) == 0 {
		return "", nil, false
	}
	return "(" + strings.Join(conditions, " OR ") + ")", args, true
}

// tableName 返回模型对应的真实表名（优先使用模型自定义的 TableName）
// @author: [bluefrog](https://github.com/freewu)
// @function: tableName
// @description: 用于拼接数据权限子查询，避免硬编码表名
// @param: model interface{} 模型指针
// @return: string
func tableName(model interface{}) string {
	if namer, ok := model.(interface{ TableName() string }); ok {
		return namer.TableName()
	}
	// 未自定义表名时按 GORM 命名策略（表前缀 + 单数）推导
	if database.DB != nil {
		return database.DB.NamingStrategy.TableName(reflect.TypeOf(model).Elem().Name())
	}
	return ""
}

// NormalizeDataScope 校验并补全角色数据范围
// @author: [bluefrog](https://github.com/freewu)
// @function: NormalizeDataScope
// @description: 未传数据范围（0）时按表默认值 1（全部数据权限）处理，取值必须在 1~5 之间
// @param: dataScope uint 数据范围
// @return: uint 补全后的数据范围, error
func NormalizeDataScope(dataScope uint) (uint, error) {
	if dataScope == 0 { // 与 sys_role.data_scope 的表默认值保持一致
		return DataScopeAll, nil
	}
	if dataScope > DataScopeSelf {
		return 0, fmt.Errorf("数据范围不正确（1全部数据权限 2自定义数据权限 3本部门数据权限 4本部门及以下数据权限 5仅本人数据权限）")
	}
	return dataScope, nil
}
