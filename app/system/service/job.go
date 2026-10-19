package service

import (
	"errors"
	"strings"
	"time"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/task"
	"github.com/bluefroog/admintpl/common/cron"
	"gorm.io/gorm"
)

type Job struct{}

// GetCount 获取定时任务列表数据数量
// @author: [bluefrog](https://github.com/freewu)
// @function: GetCount
// @description: 获取定时任务列表数据数量
// @param: searchParams *domain.JobSearchRequest
// @return: err error, total int64
func (s Job) GetCount(searchParams *domain.JobSearchRequest) (err error, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Job{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取定时任务列表数据
// @author: [bluefrog](https://github.com/freewu)
// @function: GetList
// @description: 获取定时任务列表数据
// @param: searchParams *domain.JobSearchRequest
// @return: err error, list interface{}, total int64
func (s Job) GetList(searchParams *domain.JobSearchRequest) (err error, list []domain.Job, total int64) {
	// 创建db
	db := domain.DB.Model(&domain.Job{})
	// 条件过滤
	db = s.parseFilter(db, searchParams)
	// 统计数据
	err = db.Count(&total).Error
	// 如果数据 0,也没有必要处理以下动作了
	if total > 0 {
		// 排序
		db.Order("job_id desc")
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

// @author: [bluefrog](https://github.com/freewu)
// @function: parseFilter
// @description: 定时任务列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.JobSearchRequest
// @return: *gorm.DB
func (s Job) parseFilter(db *gorm.DB, searchParams *domain.JobSearchRequest) *gorm.DB {
	// 条件过滤
	if searchParams.ID != nil { // 任务ID
		db = db.Where("job_id = ?", searchParams.ID)
	}
	if len(searchParams.IDIn) > 0 { // 任务ID 数组
		db = db.Where("job_id IN (?)", searchParams.IDIn)
	}
	if len(searchParams.IDNotIn) > 0 { // IDNotIn
		db = db.Where("job_id NOT IN (?)", searchParams.IDNotIn)
	}
	if searchParams.Name != "" { // 任务名称
		db = db.Where("job_name LIKE ?", "%"+searchParams.Name+"%")
	}
	if searchParams.Group != "" { // 任务组名
		db = db.Where("job_group = ?", searchParams.Group)
	}
	if searchParams.Executor != "" { // 任务执行器
		db = db.Where("job_executor = ?", searchParams.Executor)
	}
	if searchParams.InvokeTarget != "" { // 调用目标字符串
		db = db.Where("invoke_target LIKE ?", "%"+searchParams.InvokeTarget+"%")
	}
	if searchParams.Status != nil { // 状态（0正常 1暂停）
		db = db.Where("status = ?", searchParams.Status)
	}
	if searchParams.CreateBy != "" { // 创建者
		db = db.Where("create_by LIKE ?", "%"+searchParams.CreateBy+"%")
	}
	if searchParams.UpdateBy != "" { // 更新者
		db = db.Where("update_by LIKE ?", "%"+searchParams.UpdateBy+"%")
	}
	if searchParams.Keyword != "" { // 关键词
		k := strings.Trim(searchParams.Keyword, " \t\r\n")
		k1 := "%" + k + "%"
		db = db.Where("(`job_name` LIKE ? OR `job_group` LIKE ? OR `invoke_target` LIKE ? OR `job_id` = ? )", k1, k1, k1, k)
	}
	return db
}

// Create 添加定时任务
// @author: [bluefrog](https://github.com/freewu)
// @function: Create
// @description: 添加定时任务
// @param: data *domain.JobCreateRequest
// @return: error
func (s Job) Create(data *domain.JobCreateRequest) error {
	// 判断任务名称是否唯一
	if s.NameIsExist(data.Name, nil) {
		return errors.New("已存在相同的任务名称")
	}
	// 校验 cron 表达式、调用目标并补全默认值
	if err := s.ValidateJob(&data.JobBase); err != nil {
		return err
	}
	now := time.Now()
	// 使用 map 写入：concurrent/status/misfire_policy 的零值（如 concurrent="0" 允许并发）会被
	// GORM 的 default 标签忽略，改用 map 才能按请求值落库
	record := map[string]interface{}{
		"Name":           data.Name,
		"Group":          data.Group,
		"Executor":       data.Executor,
		"InvokeTarget":   data.InvokeTarget,
		"JobArgs":        data.JobArgs,
		"JobKwargs":      data.JobKwargs,
		"CronExpression": data.CronExpression,
		"MisfirePolicy":  data.MisfirePolicy,
		"Concurrent":     data.Concurrent,
		"Status":         data.Status,
		"Remark":         data.Remark,
		"CreateBy":       "", // todo 通过登录服务获取
		"CreateTime":     now,
		"UpdateBy":       "", // todo 通过登录服务获取
		"UpdateTime":     now,
	}
	if err := domain.DB.Model(&domain.Job{}).Create(record).Error; err != nil {
		return err
	}
	// 新增后按最新配置注册调度（状态为「暂停」时不注册）
	var created domain.Job
	if err := domain.DB.Where("job_name = ?", data.Name).First(&created).Error; err == nil {
		JobScheduler{}.Refresh(created.ID)
	}
	return nil
}

// Update 修改定时任务
// @author: [bluefrog](https://github.com/freewu)
// @function: Update
// @description: 修改定时任务
// @param: data *domain.JobUpdateRequest
// @return: err error
func (s Job) Update(data *domain.JobUpdateRequest) (err error) {
	// 判断任务名称是否唯一
	if s.NameIsExist(data.Name, &[]uint{data.ID}) {
		return errors.New("已存在相同的任务名称")
	}
	// 校验 cron 表达式、调用目标并补全默认值
	if err = s.ValidateJob(&data.JobBase); err != nil {
		return err
	}
	record := make(map[string]interface{})
	record["Name"] = data.Name
	record["Group"] = data.Group
	record["Executor"] = data.Executor
	record["InvokeTarget"] = data.InvokeTarget
	record["JobArgs"] = data.JobArgs
	record["JobKwargs"] = data.JobKwargs
	record["CronExpression"] = data.CronExpression
	record["MisfirePolicy"] = data.MisfirePolicy
	record["Concurrent"] = data.Concurrent
	record["Status"] = data.Status
	record["Remark"] = data.Remark
	record["UpdateBy"] = "" // todo 通过登录服务获取
	record["UpdateTime"] = time.Now()

	if err = s.UpdateByMap(data.ID, record); err != nil {
		return err
	}
	// 任务配置变更后重新注册调度（含启用/停用）
	JobScheduler{}.Refresh(data.ID)
	return nil
}

// UpdateByMap 更新指定ID的定时任务信息
// @author: [bluefrog](https://github.com/freewu)
// @function: UpdateByMap
// @description: 更新指定ID的定时任务信息
// @param: id uint
// @param: updateMap map[string]interface{}
// @return: err error
func (s Job) UpdateByMap(id uint, updateMap map[string]interface{}) (err error) {
	if err := domain.DB.Model(&domain.Job{}).Where("job_id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return err
}

// Delete 删除定时任务
// @author: [bluefrog](https://github.com/freewu)
// @function: Delete
// @description: 删除定时任务
// @param: ids []uint
// @return: err error
func (s Job) Delete(ids []uint) (err error) {
	// 摘除调度
	for _, id := range ids {
		JobScheduler{}.Remove(id)
	}
	// 与若依一致：删除任务时同时清理其执行日志（日志表没有 job_id，按 任务名称+任务组 匹配）
	var jobs []domain.Job
	if err = domain.DB.Where("job_id in (?) ", ids).Find(&jobs).Error; err == nil {
		for _, job := range jobs {
			domain.DB.Where("job_name = ? AND job_group = ?", job.Name, job.Group).Delete(&domain.JobLog{})
		}
	}
	if err = domain.DB.Where("job_id in (?) ", ids).Unscoped().Delete(&domain.Job{}).Error; err != nil {
		return err
	}
	return nil
}

// Detail 获取指定ID定时任务详情
// @author: [bluefrog](https://github.com/freewu)
// @function: Detail
// @description: 获取指定ID定时任务详情
// @param: id uint
// @return: *domain.Job,err error
func (s Job) Detail(id uint) (*domain.Job, error) {
	var detail domain.Job
	db := domain.DB.Where("job_id = ?", id)
	if err := db.First(&detail).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

// NameIsExist 判断任务名称是否存在
// @author: [bluefrog](https://github.com/freewu)
// @function: NameIsExist
// @description: 判断任务名称是否存在
// @param: name string 任务名称
// @param: excludeIds *[]uint 不包含的任务ID
// @return: err error
func (s Job) NameIsExist(name string, excludeIds *[]uint) bool {
	// 判断名字是否唯一
	var filter = &domain.JobSearchRequest{
		Name: name,
	}
	if excludeIds != nil {
		filter.IDNotIn = *excludeIds
	}
	_, total := s.GetCount(filter)
	return total > 0
}

// ChangeStatus 修改任务状态
// @author: [bluefrog](https://github.com/freewu)
// @function: ChangeStatus
// @description: 修改定时任务状态（0正常 1暂停）
// @param: req *domain.JobChangeStatusRequest
// @return: err error
func (s Job) ChangeStatus(req *domain.JobChangeStatusRequest) (err error) {
	if req.Status != JobStatusRunning && req.Status != JobStatusPaused {
		return errors.New("任务状态只能为 0（正常）或 1（暂停）")
	}
	record := make(map[string]interface{})
	record["Status"] = req.Status
	record["UpdateTime"] = time.Now()
	if err = s.UpdateByMap(req.ID, record); err != nil {
		return err
	}
	// 状态为「正常」时注册调度，「暂停」时摘除调度
	JobScheduler{}.Refresh(req.ID)
	return nil
}

// Run 立即执行一次任务（与任务状态无关，与若依「执行一次」一致）
// @author: [bluefrog](https://github.com/freewu)
// @function: Run
// @description: 立即执行一次任务，执行结果写入 sys_job_log
// @param: id uint
// @return: err error
func (s Job) Run(id uint) (err error) {
	return JobScheduler{}.RunOnce(id)
}

// ValidateJob 校验任务配置并补全默认值
// @author: [bluefrog](https://github.com/freewu)
// @function: ValidateJob
// @description: 校验 cron 表达式、调用目标字符串、并发策略、错过执行策略与状态
// @param: data *domain.JobBase
// @return: err error
func (s Job) ValidateJob(data *domain.JobBase) error {
	if strings.TrimSpace(data.CronExpression) == "" {
		return errors.New("Cron 表达式不能为空")
	}
	if _, err := cron.Parse(data.CronExpression); err != nil {
		return errors.New("Cron 表达式不正确：" + err.Error())
	}
	// 调用目标必须是已注册的目标（与若依的白名单校验等价，避免任意方法调用）
	if _, _, err := task.LookupTarget(data.InvokeTarget); err != nil {
		return errors.New("调用目标字符串错误：" + err.Error())
	}
	switch data.Concurrent {
	case "":
		data.Concurrent = JobConcurrentRefuse // 与表默认值保持一致
	case JobConcurrentAllow, JobConcurrentRefuse:
	default:
		return errors.New("并发策略只能为 0（允许）或 1（禁止）")
	}
	switch data.MisfirePolicy {
	case "":
		data.MisfirePolicy = JobMisfireDoNothing // 与表默认值保持一致
	case JobMisfireFireNow, JobMisfireFireOnce, JobMisfireDoNothing:
	default:
		return errors.New("错过执行策略只能为 1（立即执行）、2（执行一次）或 3（放弃执行）")
	}
	if data.Status != JobStatusRunning && data.Status != JobStatusPaused {
		return errors.New("任务状态只能为 0（正常）或 1（暂停）")
	}
	if strings.TrimSpace(data.Group) == "" {
		data.Group = "default"
	}
	if strings.TrimSpace(data.Executor) == "" {
		data.Executor = "default"
	}
	return nil
}

// CheckNameUnique 校验任务名称是否唯一
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckNameUnique
// @description: 校验任务名称是否唯一
// @param: name string 任务名称
// @param: excludeIds *[]uint 排除的任务ID
// @return: err error, unique bool
func (s Job) CheckNameUnique(name string, excludeIds *[]uint) (err error, unique bool) {
	if name == "" {
		return nil, true
	}
	db := domain.DB.Model(&domain.Job{}).Where("job_name = ?", name)
	if excludeIds != nil && len(*excludeIds) > 0 {
		db = db.Where("job_id NOT IN (?)", *excludeIds)
	}
	var count int64
	if err = db.Count(&count).Error; err != nil {
		return err, false
	}
	return nil, count == 0
}
