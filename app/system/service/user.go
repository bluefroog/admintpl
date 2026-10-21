package service

import (
	"errors"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/database"
	"github.com/bluefroog/admintpl/common/lib"
	"gorm.io/gorm"
	"strings"
	"time"
)

type User struct{}

// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取用户列表数据（按登录用户的数据权限过滤）
// @param: searchParams *domain.UserSearchRequest
// @param: loginUser *domain.LoginUser 登录用户（认证关闭时为 nil）
// @return: err error, list interface{}, total int64
func (s User) GetList(searchParams *domain.UserSearchRequest, loginUser *domain.LoginUser) (err error, list []domain.User, total int64) {
	// 创建db
	db := database.DB.Model(&domain.User{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 数据权限（部门ID列 + 用户ID列）
	db = DataScope{}.Apply(db, loginUser, DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"})
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		orderBy := "id desc"
		if searchParams.OrderBy != "" {
			db.Order(searchParams.OrderBy)
		}
		db.Order(orderBy)
		// 分页
		if searchParams.PageSize > 0 {
			limit := searchParams.PageSize
			offset := searchParams.PageSize * (searchParams.Page - 1)
			db.Limit(limit).Offset(offset)
		}
		err = db.Find(&list).Error
	}
	return err, list, total
}

// @author: [bluefrog](https://github.com/freewu)
// @function: parseFilter
// @description: 用户列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.UserSearchRequest
// @return: *gorm.DB
func (s User) parseFilter(db *gorm.DB, searchParams *domain.UserSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.Username != "" { // 用户名
		db = db.Where("username = ?", searchParams.Username)
	}
	if searchParams.Mobile != "" { // 手机号
		db = db.Where("mobile = ?", searchParams.Mobile)
	}
	if searchParams.Status != nil { // 用户状态;0:禁用,1:正常,2:未验证
		db = db.Where("status = ?", searchParams.Status)
	}
	if searchParams.IsAdmin != nil { // 是否后台管理员 1 是  0 否
		db = db.Where("is_admin = ?", searchParams.IsAdmin)
	}
	if searchParams.DepartmentID != nil { // 部门ID
		db = db.Where("department_id = ?", searchParams.DepartmentID)
	}
	if searchParams.Gender != nil { // 性别;0:保密,1:男,2:女
		db = db.Where("gender = ?", searchParams.Gender)
	}
	if searchParams.Nickname != "" { // 用户昵称
		db = db.Where("nickname = ?", searchParams.Nickname)
	}
	if searchParams.LastLoginIP != "" { // 最后登录IP
		db = db.Where("last_login_ip = ?", searchParams.LastLoginIP)
	}
	if searchParams.DateColumn == "lastLoginTime" { // 最后登录时间
		if searchParams.BeginTime != "" { // 开始时间
			db = db.Where("last_login_time >= ?", searchParams.BeginTime)
		}
		if searchParams.EndTime != "" { // 结束时间
			db = db.Where("last_login_time <= ?", searchParams.EndTime)
		}
	}
	if searchParams.DateColumn == "registerTime" { // 注册时间
		if searchParams.BeginTime != "" { // 开始时间
			db = db.Where("register_time >= ?", searchParams.BeginTime)
		}
		if searchParams.EndTime != "" { // 结束时间
			db = db.Where("register_time <= ?", searchParams.EndTime)
		}
	}
	if searchParams.Email != "" { // 登录邮箱
		db = db.Where("email LIKE ?", "%"+searchParams.Email+"%")
	}
	if searchParams.Remark != "" { // 备注
		db = db.Where("remark LIKE ?", "%"+searchParams.Remark+"%")
	}
	if searchParams.Keyword != "" { // 关键词
		k1 := strings.Trim(searchParams.Keyword, " \t\r\n")
		k := "%" + k1 + "%"
		db = db.Where("("+
			"username LIKE ? OR "+
			"mobile LIKE ? OR "+
			"email LIKE ? OR "+
			"nickname LIKE ? OR "+
			"last_login_ip LIKE ? OR "+
			"remark LIKE ? )", k, k, k, k, k, k)
	}
	return db
}

// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加用户数据，密码取参数 sys.user.initPassword（默认 admin123），并生成随机盐
// @param: data *domain.UserAddRequest
// @return: error
func (s User) Create(data *domain.UserAddRequest) error {
	// todo 判断用户名是否已存在
	// todo 判断手机号是否已存在
	// todo 判断邮箱是否已存在
	// 初始密码取系统参数 sys.user.initPassword，未配置时回退内置默认值
	initPassword := seedInitPassword
	if err, value := (Config{}).GetByKey(seedInitPasswordKey); err == nil && value != "" {
		initPassword = value
	}
	// 服务端已知明文，按「前端 hash + 后端加盐 hash」两段式写入
	password, salt, err := lib.HashPlainPassword(initPassword)
	if err != nil {
		return err
	}
	// 注意：结构体方式 Create 会让带 default 标签的零值字段（如 status 默认 1）被数据库默认值覆盖，
	// 导致显式传入的「禁用（0）」被写成「正常（1）」，因此与 Update 保持一致，使用 map 显式写入。
	now := time.Now()
	record := map[string]interface{}{
		"Username":     data.Username,
		"Mobile":       data.Mobile,
		"Nickname":     data.Nickname,
		"Birthday":     data.Birthday,
		"Status":       data.Status,
		"Email":        data.Email,
		"Gender":       data.Gender,
		"Avatar":       data.Avatar,
		"DepartmentID": data.DepartmentID,
		"Remark":       data.Remark,
		"IsAdmin":      data.IsAdmin,
		"Password":     password, // 初始密码（两段式 hash）
		"Salt":         salt,     // 密码盐
		"RegisterTime": now,      // 注册时间
		"CreatedAt":    now,
		"UpdatedAt":    now,
	}

	return database.DB.Model(&domain.User{}).Create(record).Error
}

// @author: [bluefrog](https://github.com/freewu)
// @function: Update
// @description: 修改用户
// @param: data *model.UserEditRequest
// @return: err error
func (s User) Update(data *domain.UserEditRequest) (err error) {
	// todo 判断用户名是否已存在
	// todo 判断手机号是否已存在
	// todo 判断邮箱是否已存在

	record := make(map[string]interface{})
	record["Username"] = data.Username
	record["Mobile"] = data.Mobile
	record["Nickname"] = data.Nickname
	record["Birthday"] = data.Birthday
	record["Status"] = data.Status
	record["Email"] = data.Email
	record["Gender"] = data.Gender
	record["Avatar"] = data.Avatar
	record["DepartmentID"] = data.DepartmentID
	record["Remark"] = data.Remark
	record["IsAdmin"] = data.IsAdmin

	return s.UpdateByMap(data.ID, record)
}

// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateByMap
// @description: 更新指定ID的用户信息
// @param: id uint
// @param: updateMap map[string]interface{}
// @return: err error
func (s User) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := database.DB.Model(&domain.User{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 删除用户，同时清理用户与角色/岗位的关联记录（避免遗留孤儿数据）
// @param: ids []uint
// @return: err error
func (s User) Delete(ids []uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	// 删除用户与角色/岗位的关联记录（避免遗留孤儿数据）
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id in (?)", ids).Unscoped().Delete(&domain.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id in (?)", ids).Unscoped().Delete(&domain.UserPost{}).Error; err != nil {
			return err
		}
		return tx.Where("id in (?)", ids).Unscoped().Delete(&domain.User{}).Error
	})
}

// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID用户详情
// @param: id uint
// @return: *domain.User,err error
func (s User) Detail(id uint) (*domain.User, error) {
	var detail domain.User
	db := database.DB.Where("id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// GetByUsername 根据登录账号获取用户（登录认证使用）
// @author: [bluefrog](https://github.com/freewu)
// @function: GetByUsername
// @description: 根据登录账号获取用户详情
// @param: username string 登录账号
// @return: *domain.User, error
func (s User) GetByUsername(username string) (*domain.User, error) {
	var detail domain.User
	if err := database.DB.Where("username = ?", username).First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// CheckUniqueUsername 校验用户名是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckUniqueUsername
// @description: 校验用户名是否唯一
// @param: req *domain.UserCheckUniqueRequest
// @return: err error, unique bool
func (s User) CheckUniqueUsername(req *domain.UserCheckUniqueRequest) (err error, unique bool) {
	if req.Username == "" {
		return nil, true
	}
	db := database.DB.Model(&domain.User{}).Where("username = ?", req.Username)
	if req.ID != nil {
		db = db.Where("id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}

// CheckUniqueMobile 校验手机号是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckUniqueMobile
// @description: 校验手机号是否唯一
// @param: req *domain.UserCheckUniqueRequest
// @return: err error, unique bool
func (s User) CheckUniqueMobile(req *domain.UserCheckUniqueRequest) (err error, unique bool) {
	if req.Mobile == "" {
		return nil, true
	}
	db := database.DB.Model(&domain.User{}).Where("mobile = ?", req.Mobile)
	if req.ID != nil {
		db = db.Where("id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}

// CheckUniqueEmail 校验邮箱是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckUniqueEmail
// @description: 校验邮箱是否唯一
// @param: req *domain.UserCheckUniqueRequest
// @return: err error, unique bool
func (s User) CheckUniqueEmail(req *domain.UserCheckUniqueRequest) (err error, unique bool) {
	if req.Email == "" {
		return nil, true
	}
	db := database.DB.Model(&domain.User{}).Where("email = ?", req.Email)
	if req.ID != nil {
		db = db.Where("id <> ?", *req.ID)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}

// ResetPwd 重置用户密码
// @author: [bluefrog](https://github.com/freewu)
// @function: ResetPwd
// @description: 重置指定用户密码
// @param: req *domain.UserResetPwdRequest
// @return: err error
func (s User) ResetPwd(req *domain.UserResetPwdRequest) (err error) {
	// 前端已对明文做过 hash，这里再加盐做后端 hash
	password, salt, err := lib.HashPassword(req.Password)
	if err != nil {
		return err
	}
	record := make(map[string]interface{})
	record["Password"] = password
	record["Salt"] = salt
	record["UpdatedAt"] = time.Now()
	return s.UpdateByMap(req.ID, record)
}

// ChangeStatus 修改用户状态
// @author: [bluefrog](https://github.com/freewu)
// @function: ChangeStatus
// @description: 修改用户状态
// @param: req *domain.UserChangeStatusRequest
// @return: err error
func (s User) ChangeStatus(req *domain.UserChangeStatusRequest) (err error) {
	record := make(map[string]interface{})
	record["Status"] = req.Status
	record["UpdatedAt"] = time.Now()
	return s.UpdateByMap(req.ID, record)
}

// AssignRoles 给用户分配角色（先删除原有关联，再批量新增）
// @author: [bluefrog](https://github.com/freewu)
// @function: AssignRoles
// @description: 用户角色授权
// @param: req *domain.UserAssignRoleRequest
// @return: err error
func (s User) AssignRoles(req *domain.UserAssignRoleRequest) (err error) {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", req.ID).Delete(&domain.UserRole{}).Error; err != nil {
			return err
		}
		if len(req.RoleIDs) == 0 {
			return nil
		}
		userRoles := make([]domain.UserRole, 0, len(req.RoleIDs))
		for _, roleId := range req.RoleIDs {
			if roleId == 0 {
				continue
			}
			userRoles = append(userRoles, domain.UserRole{UserID: req.ID, RoleID: roleId})
		}
		if len(userRoles) == 0 {
			return nil
		}
		return tx.Create(&userRoles).Error
	})
}

// AssignPosts 给用户分配岗位（先删除原有关联，再批量新增）
// @author: [bluefrog](https://github.com/freewu)
// @function: AssignPosts
// @description: 用户岗位授权
// @param: req *domain.UserAssignPostRequest
// @return: err error
func (s User) AssignPosts(req *domain.UserAssignPostRequest) (err error) {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", req.ID).Delete(&domain.UserPost{}).Error; err != nil {
			return err
		}
		if len(req.PostIDs) == 0 {
			return nil
		}
		userPosts := make([]domain.UserPost, 0, len(req.PostIDs))
		for _, postId := range req.PostIDs {
			if postId == 0 {
				continue
			}
			userPosts = append(userPosts, domain.UserPost{UserID: req.ID, PostID: postId})
		}
		if len(userPosts) == 0 {
			return nil
		}
		return tx.Create(&userPosts).Error
	})
}

// GetUserRoles 获取用户已分配的角色ID列表
// @author: [bluefrog](https://github.com/freewu)
// @function: GetUserRoles
// @description: 获取用户已分配的角色ID列表
// @param: userId uint
// @return: err error, roleIds []uint
func (s User) GetUserRoles(userId uint) (err error, roleIds []uint) {
	if err = database.DB.Model(&domain.UserRole{}).Where("user_id = ?", userId).Pluck("role_id", &roleIds).Error; err != nil {
		return err, nil
	}
	return nil, roleIds
}

// GetUserPosts 获取用户已分配的岗位ID列表
// @author: [bluefrog](https://github.com/freewu)
// @function: GetUserPosts
// @description: 获取用户已分配的岗位ID列表
// @param: userId uint
// @return: err error, postIds []uint
func (s User) GetUserPosts(userId uint) (err error, postIds []uint) {
	if err = database.DB.Model(&domain.UserPost{}).Where("user_id = ?", userId).Pluck("post_id", &postIds).Error; err != nil {
		return err, nil
	}
	return nil, postIds
}

// HasRole 判断用户是否拥有指定角色
// @author: [bluefrog](https://github.com/freewu)
// @function: HasRole
// @description: 判断用户是否拥有指定角色
// @param: userId uint
// @param: roleId uint
// @return: err error, has bool
func (s User) HasRole(userId uint, roleId uint) (err error, has bool) {
	var count int64
	if err = database.DB.Model(&domain.UserRole{}).Where("user_id = ? AND role_id = ?", userId, roleId).Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count > 0
}

// UpdateProfile 修改用户个人资料
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateProfile
// @description: 修改用户个人资料
// @param: req *domain.UserProfileUpdateRequest
// @return: err error
func (s User) UpdateProfile(req *domain.UserProfileUpdateRequest) (err error) {
	record := make(map[string]interface{})
	record["Nickname"] = req.Nickname
	record["Email"] = req.Email
	record["Mobile"] = req.Mobile
	record["Gender"] = req.Gender
	record["UpdatedAt"] = time.Now()
	return s.UpdateByMap(req.ID, record)
}

// UpdatePassword 修改用户密码（校验旧密码）
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdatePassword
// @description: 修改用户密码
// @param: req *domain.UserPasswordUpdateRequest
// @return: err error
func (s User) UpdatePassword(req *domain.UserPasswordUpdateRequest) (err error) {
	detail, err := s.Detail(req.ID)
	if err != nil {
		return errors.New("用户不存在")
	}
	if !lib.CheckPassword(detail.Password, detail.Salt, req.OldPassword) {
		return errors.New("原密码错误")
	}
	// 修改密码时重新生成盐
	password, salt, err := lib.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	record := make(map[string]interface{})
	record["Password"] = password
	record["Salt"] = salt
	record["UpdatedAt"] = time.Now()
	return s.UpdateByMap(req.ID, record)
}

// UpdateAvatar 修改用户头像
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateAvatar
// @description: 修改用户头像
// @param: req *domain.UserAvatarUpdateRequest
// @return: err error
func (s User) UpdateAvatar(req *domain.UserAvatarUpdateRequest) (err error) {
	record := make(map[string]interface{})
	record["Avatar"] = req.Avatar
	record["UpdatedAt"] = time.Now()
	return s.UpdateByMap(req.ID, record)
}
