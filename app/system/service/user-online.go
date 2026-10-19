package service

import (
	"strings"
	"time"

	"github.com/bluefroog/admintpl/app/system/domain"

	"gorm.io/gorm"
)

type UserOnline struct{}

// GetCount 获取在线用户列表数据数量
// @author: [bluefrog](https://github.com/freewu)
// @function: GetCount
// @description: 获取在线用户列表数据数量
// @param: searchParams *domain.UserOnlineSearchRequest
// @return: err error, total int64
func (s UserOnline) GetCount(searchParams *domain.UserOnlineSearchRequest) (err error, total int64) {
	db := domain.DB.Model(&domain.UserOnline{})
	db = s.parseFilter(db, searchParams)
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取在线用户列表数据
// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取在线用户列表数据
// @param: searchParams *domain.UserOnlineSearchRequest
// @return: err error, list []domain.UserOnline, total int64
func (s UserOnline) GetList(searchParams *domain.UserOnlineSearchRequest) (err error, list []domain.UserOnline, total int64) {
	db := domain.DB.Model(&domain.UserOnline{})
	db = s.parseFilter(db, searchParams)
	if err = db.Count(&total).Error; err != nil {
		return err, nil, 0
	}
	if total > 0 {
		db.Order("id desc")
		if searchParams.OrderBy != "" {
			db.Order(searchParams.OrderBy)
		}
		if searchParams.PageSize > 0 {
			limit := searchParams.PageSize
			offset := searchParams.PageSize * (searchParams.Page - 1)
			db.Limit(limit).Offset(offset)
		}
		err = db.Find(&list).Error
	}
	return err, list, total
}

// parseFilter 在线用户列表搜索条件过滤
// @author: [bluefrog](https://github.com/freewu)
// @function: parseFilter
// @description: 在线用户列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.UserOnlineSearchRequest
// @return: *gorm.DB
func (s UserOnline) parseFilter(db *gorm.DB, searchParams *domain.UserOnlineSearchRequest) *gorm.DB {
	if searchParams.ID != nil {
		db = db.Where("id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 {
		db = db.Where("id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 {
		db = db.Where("id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.UUID != "" {
		db = db.Where("uuid = ?", searchParams.UUID)
	}
	if searchParams.Token != "" {
		db = db.Where("token = ?", searchParams.Token)
	}
	if searchParams.UserName != "" {
		db = db.Where("user_name = ?", searchParams.UserName)
	}
	if searchParams.IP != "" {
		db = db.Where("ip = ?", searchParams.IP)
	}
	if searchParams.Explorer != "" {
		db = db.Where("explorer = ?", searchParams.Explorer)
	}
	if searchParams.OS != "" {
		db = db.Where("os = ?", searchParams.OS)
	}
	if searchParams.Keyword != "" {
		k := strings.Trim(searchParams.Keyword, " \t\r\n")
		k1 := "%" + k + "%"
		db = db.Where("(`user_name` LIKE ? OR `ip` LIKE ? OR `explorer` LIKE ? OR `os` LIKE ?)", k1, k1, k1, k1)
	}
	return db
}

// Create 添加在线用户记录
// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加在线用户记录
// @param: data *domain.UserOnlineCreateRequest
// @return: error
func (s UserOnline) Create(data *domain.UserOnlineCreateRequest) error {
	record := new(domain.UserOnline)
	record.UUID = data.UUID
	record.Token = data.Token
	record.UserName = data.UserName
	record.IP = data.IP
	record.Explorer = data.Explorer
	record.OS = data.OS
	record.CreateTime = uint(time.Now().UnixMilli())
	return domain.DB.Create(&record).Error
}

// DeleteByToken 根据用户token删除在线记录（用户退出登录时调用）
// @author: [bluefrog](https://github.com/freewu)
// @function: DeleteByToken
// @description: 根据用户token删除在线记录
// @param: token string 用户token
// @return: err error
func (s UserOnline) DeleteByToken(token string) (err error) {
	if token == "" {
		return nil
	}
	return domain.DB.Where("token = ?", token).Unscoped().Delete(&domain.UserOnline{}).Error
}

// Delete 批量删除在线用户记录
// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 批量删除在线用户记录
// @param: ids []uint
// @return: err error
func (s UserOnline) Delete(ids []uint) (err error) {
	if len(ids) == 0 {
		return nil
	}
	return domain.DB.Where("id in (?)", ids).Unscoped().Delete(&domain.UserOnline{}).Error
}

// DeleteNotInIds 删除不在指定ID列表中的在线记录（清理失效会话）
// @author: [bluefrog](https://github.com/freewu)
// @function: DeleteNotInIds
// @description: 删除不在指定ID列表中的在线记录
// @param: ids []uint
// @return: err error
func (s UserOnline) DeleteNotInIds(ids []uint) (err error) {
	db := domain.DB.Session(&gorm.Session{AllowGlobalUpdate: true})
	if len(ids) > 0 {
		db = db.Where("id NOT IN (?)", ids)
	}
	return db.Unscoped().Delete(&domain.UserOnline{}).Error
}

// ForceLogout 强退在线用户
// @author: [bluefrog](https://github.com/freewu)
// @function: ForceLogout
// @description: 根据用户token强退在线用户
// @param: req *domain.UserOnlineForceLogoutRequest
// @return: err error
func (s UserOnline) ForceLogout(req *domain.UserOnlineForceLogoutRequest) (err error) {
	return s.DeleteByToken(req.Token)
}

// Detail 获取指定ID在线用户详情
// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID在线用户详情
// @param: id uint
// @return: *domain.UserOnline, err error
func (s UserOnline) Detail(id uint) (*domain.UserOnline, error) {
	var detail domain.UserOnline
	if err := domain.DB.Where("id = ?", id).First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// GetByToken 根据令牌获取在线用户记录（登录认证使用）
// @author: [bluefrog](https://github.com/freewu)
// @function: GetByToken
// @description: 根据令牌获取在线用户记录
// @param: token string 用户令牌
// @return: *domain.UserOnline, error
func (s UserOnline) GetByToken(token string) (*domain.UserOnline, error) {
	var detail domain.UserOnline
	if err := domain.DB.Where("token = ?", token).First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}
