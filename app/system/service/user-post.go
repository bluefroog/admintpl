package service

import (
	"gorm.io/gorm"
	"github.com/bluefroog/admintpl/app/system/domain"
)

type UserPost struct{}

// GetCount 获取用户岗位关联列表数据数量
//@author: [bluefrog](https://github.com/freewu)
//@function: GetCount
//@description: 获取用户岗位关联列表数据数量
//@param: searchParams *domain.UserPostSearchRequest
//@return: err error, total int64
func (s UserPost) GetCount(searchParams *domain.UserPostSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.UserPost{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取用户岗位关联列表数据
//@author: [bluefrog](https://github.com/freewu)
//@function: GetList
//@description: 获取用户岗位关联列表数据
//@param: searchParams *domain.UserPostSearchRequest
//@return: err error, list interface{}, total int64
func (s UserPost) GetList(searchParams *domain.UserPostSearchRequest) (err error, list []domain.UserPost, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.UserPost{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("user_id asc")
		if searchParams.OrderBy != "" {
			db.Order(searchParams.OrderBy)
		}
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

//@author: [bluefrog](https://github.com/freewu)
//@function: parseFilter
//@description: 用户岗位关联列表搜索条件过滤
//@param: db *gorm.DB
//@param: searchParams *domain.UserPostSearchRequest
//@return: *gorm.DB
func (s UserPost) parseFilter(db *gorm.DB, searchParams *domain.UserPostSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.UserID != nil { // 用户ID
		db = db.Where("user_id = ?", searchParams.UserID)
	}
	if len(searchParams.UserIDIn) > 0 { // 用户ID 数组
		db = db.Where("user_id IN (?)", searchParams.UserIDIn)
	}
	if searchParams.PostID != nil { // 岗位ID
		db = db.Where("post_id = ?", searchParams.PostID)
	}
	if len(searchParams.PostIDIn) > 0 { // 岗位ID 数组
		db = db.Where("post_id IN (?)", searchParams.PostIDIn)
	}
	return db
}

// Create 添加用户岗位关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Create
//@description: 添加用户岗位关联
//@param: data *domain.UserPostCreateRequest
//@return: error
func (s UserPost) Create(data *domain.UserPostCreateRequest) error {
	userPost := new(domain.UserPost)
	userPost.UserID = data.UserID
	userPost.PostID = data.PostID

	return domain.DB.Create(&userPost).Error
}

// Delete 删除用户岗位关联
//@author: [bluefrog](https://github.com/freewu)
//@function: Delete
//@description: 删除用户岗位关联
//@param: data *domain.UserPostDeleteRequest
//@return: err error
func (s UserPost) Delete(data *domain.UserPostDeleteRequest) (err error) {
	db := domain.DB.Model(&domain.UserPost{})
	// 条件过滤
	if data.UserID != nil { // 用户ID
		db = db.Where("user_id = ?", data.UserID)
	}
	if data.PostID != nil { // 岗位ID
		db = db.Where("post_id = ?", data.PostID)
	}
	if err = db.Unscoped().Delete(&domain.UserPost{}).Error; err != nil {
		return err
	}
	return nil
}
