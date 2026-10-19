package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluefroog/admintpl/app/gen/domain"
	"github.com/bluefroog/admintpl/common/database"
	"gorm.io/gorm"
)

// Table 代码生成业务表（gen_table / gen_table_column）服务
type Table struct{}

// dbTableSelect 数据库表列表查询字段（显式别名为小写：information_schema 返回的列名为大写，会导致 GORM 无法映射到结构体）
const dbTableSelect = "table_name as table_name, table_comment as table_comment, create_time as create_time, update_time as update_time"

// dbColumnSelect 数据库列信息查询字段（同样需要显式别名）
const dbColumnSelect = "column_name as column_name, column_comment as column_comment, column_type as column_type, " +
	"ifnull((case when (is_nullable = 'no' and column_key <> 'PRI') then '1' else '0' end), '0') as is_required, " +
	"ifnull((case when column_key = 'PRI' then '1' else '0' end), '0') as is_pk, " +
	"ifnull((case when extra = 'auto_increment' then '1' else '0' end), '0') as is_increment, " +
	"ordinal_position as sort"

// GetCount 获取代码生成表列表数据数量
// @function: GetCount
// @description: 获取代码生成表列表数据数量
// @param: searchParams *domain.TableSearchRequest
// @return: err error, total int64
func (s Table) GetCount(searchParams *domain.TableSearchRequest) (err error, total int64) {
	db := s.parseFilter(database.DB.Model(&domain.Table{}), searchParams)
	err = db.Count(&total).Error
	return err, total
}

// GetList 获取代码生成表列表数据
// @function: GetList
// @description: 获取代码生成表列表数据
// @param: searchParams *domain.TableSearchRequest
// @return: err error, list []domain.Table, total int64
func (s Table) GetList(searchParams *domain.TableSearchRequest) (err error, list []domain.Table, total int64) {
	db := s.parseFilter(database.DB.Model(&domain.Table{}), searchParams)
	err = db.Count(&total).Error
	if total > 0 {
		db.Order("table_id desc")
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

// parseFilter 代码生成表列表搜索条件过滤
// @function: parseFilter
// @description: 代码生成表列表搜索条件过滤
// @param: db *gorm.DB
// @param: searchParams *domain.TableSearchRequest
// @return: *gorm.DB
func (s Table) parseFilter(db *gorm.DB, searchParams *domain.TableSearchRequest) *gorm.DB {
	if searchParams.ID != nil {
		db = db.Where("table_id = ?", *searchParams.ID)
	}
	if searchParams.TableName != "" {
		db = db.Where("table_name LIKE ?", "%"+searchParams.TableName+"%")
	}
	if searchParams.TableComment != "" {
		db = db.Where("table_comment LIKE ?", "%"+searchParams.TableComment+"%")
	}
	if searchParams.BeginTime != "" {
		db = db.Where("create_time >= ?", searchParams.BeginTime+" 00:00:00")
	}
	if searchParams.EndTime != "" {
		db = db.Where("create_time <= ?", searchParams.EndTime+" 23:59:59")
	}
	if searchParams.CreateBy != "" {
		db = db.Where("create_by LIKE ?", "%"+searchParams.CreateBy+"%")
	}
	if searchParams.UpdateBy != "" {
		db = db.Where("update_by LIKE ?", "%"+searchParams.UpdateBy+"%")
	}
	return db
}

// dbTableQuery 构造「未导入的数据库表」查询条件
// @function: dbTableQuery
// @description: 查询 information_schema.tables，过滤 qrtz_/gen_ 前缀表与已导入表
// @param: searchParams *domain.DbTableSearchRequest
// @return: *gorm.DB
func (s Table) dbTableQuery(searchParams *domain.DbTableSearchRequest) *gorm.DB {
	db := database.DB.Table("information_schema.tables").
		Where("table_schema = (select database())").
		Where("table_type = 'BASE TABLE'").
		Where("table_name NOT LIKE 'qrtz\\_%' AND table_name NOT LIKE 'gen\\_%'").
		Where("table_name NOT IN (select table_name from `" + physicalTable("gen_table") + "`)")
	if searchParams != nil {
		if searchParams.TableName != "" {
			db = db.Where("table_name LIKE ?", "%"+searchParams.TableName+"%")
		}
		if searchParams.TableComment != "" {
			db = db.Where("table_comment LIKE ?", "%"+searchParams.TableComment+"%")
		}
		if searchParams.BeginTime != "" {
			db = db.Where("create_time >= ?", searchParams.BeginTime+" 00:00:00")
		}
		if searchParams.EndTime != "" {
			db = db.Where("create_time <= ?", searchParams.EndTime+" 23:59:59")
		}
	}
	return db
}

// DbTableCount 获取未导入的数据库表数量
// @function: DbTableCount
// @description: 获取未导入的数据库表数量
// @param: searchParams *domain.DbTableSearchRequest
// @return: err error, total int64
func (s Table) DbTableCount(searchParams *domain.DbTableSearchRequest) (err error, total int64) {
	err = s.dbTableQuery(searchParams).Count(&total).Error
	return err, total
}

// DbTableList 获取未导入的数据库表列表
// @function: DbTableList
// @description: 获取未导入的数据库表列表
// @param: searchParams *domain.DbTableSearchRequest
// @return: err error, list []domain.DbTable, total int64
func (s Table) DbTableList(searchParams *domain.DbTableSearchRequest) (err error, list []domain.DbTable, total int64) {
	err, total = s.DbTableCount(searchParams)
	if err != nil || total == 0 {
		return err, list, total
	}
	db := s.dbTableQuery(searchParams).
		Select(dbTableSelect).
		Order("create_time desc")
	if searchParams.PageSize > 0 {
		db = db.Limit(searchParams.PageSize).Offset(searchParams.PageSize * (searchParams.Page - 1))
	}
	err = db.Find(&list).Error
	return err, list, total
}

// DbTableListByNames 根据表名批量获取数据库表信息
// @function: DbTableListByNames
// @description: 根据表名批量获取数据库表信息
// @param: tableNames []string 表名称列表
// @return: err error, list []domain.DbTable
func (s Table) DbTableListByNames(tableNames []string) (err error, list []domain.DbTable) {
	if len(tableNames) == 0 {
		return nil, list
	}
	err = database.DB.Table("information_schema.tables").
		Select(dbTableSelect).
		Where("table_schema = (select database())").
		Where("table_name IN (?)", tableNames).
		Find(&list).Error
	return err, list
}

// DbColumns 查询指定表的列信息
// @function: DbColumns
// @description: 查询 information_schema.columns 获取列名称、类型、主键/自增/必填标记
// @param: tableName string 表名称
// @return: err error, list []domain.DbColumn
func (s Table) DbColumns(tableName string) (err error, list []domain.DbColumn) {
	err = database.DB.Table("information_schema.columns").
		Select(dbColumnSelect).
		Where("table_schema = (select database())").
		Where("table_name = ?", tableName).
		Order("ordinal_position").
		Find(&list).Error
	return err, list
}

// Columns 获取指定表已导入的列信息
// @function: Columns
// @description: 获取指定表已导入的列信息
// @param: tableID uint 表编号
// @return: err error, list []domain.TableColumn
func (s Table) Columns(tableID uint) (err error, list []domain.TableColumn) {
	err = database.DB.Where("table_id = ?", tableID).Order("sort asc").Find(&list).Error
	return err, list
}

// Detail 获取指定编号代码生成表详情（含列信息）
// @function: Detail
// @description: 获取指定编号代码生成表详情，同时回填列信息、主键列与树表参数
// @param: tableID uint 表编号
// @return: *domain.Table, err error
func (s Table) Detail(tableID uint) (*domain.Table, error) {
	var table domain.Table
	if err := database.DB.Where("table_id = ?", tableID).First(&table).Error; err != nil {
		return nil, err
	}
	err, columns := s.Columns(tableID)
	if err != nil {
		return nil, err
	}
	table.Columns = columns
	SetPkColumn(&table)
	ParseOptions(&table)
	return &table, nil
}

// Import 导入表结构（保留表结构信息 + 初始化列属性字段）
// @function: Import
// @description: 按表名导入数据库表到 gen_table/gen_table_column，已导入的表自动跳过
// @param: tableNames []string 表名称列表
// @param: operName string 操作人
// @return: imported int 本次导入数量, err error
func (s Table) Import(tableNames []string, operName string) (imported int, err error) {
	names := make([]string, 0, len(tableNames))
	for _, name := range tableNames {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return 0, errors.New("请选择要导入的表")
	}
	// 过滤已导入的表
	var exists []string
	if err = database.DB.Model(&domain.Table{}).Where("table_name IN (?)", names).Pluck("table_name", &exists).Error; err != nil {
		return 0, err
	}
	existMap := make(map[string]bool, len(exists))
	for _, name := range exists {
		existMap[name] = true
	}
	pending := make([]string, 0, len(names))
	for _, name := range names {
		if !existMap[name] {
			pending = append(pending, name)
		}
	}
	if len(pending) == 0 {
		return 0, errors.New("选择的表已全部导入，请勿重复导入")
	}
	err, tables := s.DbTableListByNames(pending)
	if err != nil {
		return 0, err
	}
	if len(tables) == 0 {
		return 0, errors.New("未查询到可导入的表，请检查表名称是否正确")
	}
	// 事务导入，避免只写入部分数据
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for index := range tables {
			if err := s.importTable(tx, tables[index], operName); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(tables), nil
}

// importTable 导入单张表（表信息 + 列信息）
// @function: importTable
// @description: 导入单张表
// @param: tx *gorm.DB 事务
// @param: dbTable domain.DbTable 数据库表信息
// @param: operName string 操作人
// @return: err error
func (s Table) importTable(tx *gorm.DB, dbTable domain.DbTable, operName string) error {
	err, dbColumns := s.DbColumns(dbTable.TableName)
	if err != nil {
		return err
	}
	if len(dbColumns) == 0 {
		return fmt.Errorf("表 %s 未查询到列信息", dbTable.TableName)
	}
	table := domain.Table{}
	table.Name = dbTable.TableName
	table.TableComment = dbTable.TableComment
	InitTable(&table, operName)
	if err := tx.Create(&table).Error; err != nil {
		return err
	}
	columns := make([]domain.TableColumn, 0, len(dbColumns))
	for index := range dbColumns {
		column := domain.TableColumn{}
		column.TableID = table.ID
		column.ColumnName = dbColumns[index].ColumnName
		column.ColumnComment = dbColumns[index].ColumnComment
		column.ColumnType = dbColumns[index].ColumnType
		column.IsRequired = dbColumns[index].IsRequired
		column.IsPk = dbColumns[index].IsPk
		column.IsIncrement = dbColumns[index].IsIncrement
		column.Sort = dbColumns[index].Sort
		InitColumnField(&column, &table)
		columns = append(columns, column)
	}
	return tx.Create(&columns).Error
}

// SynchDb 同步数据库表结构（新增列、删除已移除列、列类型变化时重置列配置）
// @function: SynchDb
// @description: 同步数据库表结构
// @param: tableID uint 表编号
// @param: operName string 操作人
// @return: err error
func (s Table) SynchDb(tableID uint, operName string) error {
	table, err := s.Detail(tableID)
	if err != nil {
		return err
	}
	err, dbColumns := s.DbColumns(table.Name)
	if err != nil {
		return err
	}
	if len(dbColumns) == 0 {
		return fmt.Errorf("表 %s 未查询到列信息", table.Name)
	}
	// 数据库列信息索引
	dbColumnMap := make(map[string]domain.DbColumn, len(dbColumns))
	for index := range dbColumns {
		dbColumnMap[dbColumns[index].ColumnName] = dbColumns[index]
	}
	// 已导入列信息索引
	existMap := make(map[string]domain.TableColumn, len(table.Columns))
	for index := range table.Columns {
		existMap[table.Columns[index].ColumnName] = table.Columns[index]
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 表描述发生变化时同步
		if comment := s.tableComment(table.Name); comment != "" && comment != table.TableComment {
			if err := tx.Model(&domain.Table{}).Where("table_id = ?", table.ID).
				Updates(map[string]interface{}{"TableComment": comment, "UpdateBy": operName, "UpdateTime": time.Now()}).Error; err != nil {
				return err
			}
		}
		// 新增或更新列
		for index := range dbColumns {
			dbColumn := dbColumns[index]
			exist, ok := existMap[dbColumn.ColumnName]
			if !ok {
				// 新增列
				column := domain.TableColumn{}
				column.TableID = table.ID
				column.ColumnName = dbColumn.ColumnName
				column.ColumnComment = dbColumn.ColumnComment
				column.ColumnType = dbColumn.ColumnType
				column.IsRequired = dbColumn.IsRequired
				column.IsPk = dbColumn.IsPk
				column.IsIncrement = dbColumn.IsIncrement
				column.Sort = dbColumn.Sort
				InitColumnField(&column, table)
				column.CreateBy = operName
				column.UpdateBy = operName
				if err := tx.Create(&column).Error; err != nil {
					return err
				}
				continue
			}
			if exist.ColumnType != dbColumn.ColumnType {
				// 列类型变化：重置列属性字段（保留用户配置的显示类型与查询方式）
				column := domain.TableColumn{}
				column.ColumnName = dbColumn.ColumnName
				column.ColumnType = dbColumn.ColumnType
				column.IsPk = dbColumn.IsPk
				column.IsIncrement = dbColumn.IsIncrement
				column.Sort = dbColumn.Sort
				InitColumnField(&column, table)
				record := map[string]interface{}{
					"ColumnComment": dbColumn.ColumnComment,
					"ColumnType":    dbColumn.ColumnType,
					"GoType":        column.GoType,
					"GoField":       column.GoField,
					"HtmlField":     column.HtmlField,
					"HtmlType":      exist.HtmlType,
					"IsPk":          column.IsPk,
					"IsIncrement":   column.IsIncrement,
					"IsRequired":    column.IsRequired,
					"IsInsert":      column.IsInsert,
					"IsEdit":        column.IsEdit,
					"IsList":        column.IsList,
					"IsQuery":       column.IsQuery,
					"Sort":          column.Sort,
					"UpdateBy":      operName,
					"UpdateTime":    time.Now(),
				}
				if err := tx.Model(&domain.TableColumn{}).Where("column_id = ?", exist.ID).Updates(record).Error; err != nil {
					return err
				}
				continue
			}
			if exist.ColumnComment != dbColumn.ColumnComment || exist.Sort != dbColumn.Sort {
				record := map[string]interface{}{
					"ColumnComment": dbColumn.ColumnComment,
					"Sort":          dbColumn.Sort,
					"UpdateBy":      operName,
					"UpdateTime":    time.Now(),
				}
				if err := tx.Model(&domain.TableColumn{}).Where("column_id = ?", exist.ID).Updates(record).Error; err != nil {
					return err
				}
			}
		}
		// 删除数据库中已不存在的列
		removed := make([]uint, 0)
		for index := range table.Columns {
			if _, ok := dbColumnMap[table.Columns[index].ColumnName]; !ok {
				removed = append(removed, table.Columns[index].ID)
			}
		}
		if len(removed) > 0 {
			if err := tx.Where("column_id IN (?)", removed).Unscoped().Delete(&domain.TableColumn{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// tableComment 查询数据库表的描述
// @function: tableComment
// @description: 查询 information_schema.tables 中的表描述
// @param: tableName string 表名称
// @return: string
func (s Table) tableComment(tableName string) string {
	var comment string
	err := database.DB.Table("information_schema.tables").
		Select("table_comment as table_comment").
		Where("table_schema = (select database())").
		Where("table_name = ?", tableName).
		Scan(&comment).Error
	if err != nil {
		return ""
	}
	return comment
}

// Update 修改代码生成业务（表信息 + 列信息）
// @function: Update
// @description: 修改代码生成业务
// @param: data *domain.TableUpdateRequest
// @param: operName string 操作人
// @return: err error
func (s Table) Update(data *domain.TableUpdateRequest, operName string) (err error) {
	table, err := s.Detail(data.ID)
	if err != nil {
		return err
	}
	record := map[string]interface{}{
		// 注意：字段名为 Name（TableName 已被 GORM 的 TableName() 方法占用），map 键必须是字段名
		"Name":           data.TableName,
		"TableComment":   data.TableComment,
		"ClassName":      data.ClassName,
		"TplCategory":    data.TplCategory,
		"PackageName":    data.PackageName,
		"ModuleName":     data.ModuleName,
		"BusinessName":   data.BusinessName,
		"FunctionName":   data.FunctionName,
		"FunctionAuthor": data.FunctionAuthor,
		"GenType":        data.GenType,
		"GenPath":        data.GenPath,
		"Options":        BuildOptions(data),
		"Remark":         data.Remark,
		"UpdateBy":       operName,
		"UpdateTime":     time.Now(),
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Table{}).Where("table_id = ?", data.ID).Updates(record).Error; err != nil {
			return err
		}
		if len(data.Columns) == 0 {
			return nil
		}
		// 只允许修改归属于当前表的列
		existIds := make(map[uint]bool, len(table.Columns))
		for index := range table.Columns {
			existIds[table.Columns[index].ID] = true
		}
		for index := range data.Columns {
			column := data.Columns[index]
			if !existIds[column.ID] {
				continue
			}
			columnRecord := map[string]interface{}{
				"ColumnComment": column.ColumnComment,
				"GoType":        column.GoType,
				"GoField":       column.GoField,
				"HtmlField":     column.HtmlField,
				"DictType":      column.DictType,
				"QueryType":     column.QueryType,
				"HtmlType":      column.HtmlType,
				"IsRequired":    column.IsRequired,
				"IsInsert":      column.IsInsert,
				"IsEdit":        column.IsEdit,
				"IsList":        column.IsList,
				"IsQuery":       column.IsQuery,
				"Sort":          column.Sort,
				"UpdateBy":      operName,
				"UpdateTime":    time.Now(),
			}
			if err := tx.Model(&domain.TableColumn{}).Where("column_id = ? AND table_id = ?", column.ID, data.ID).
				Updates(columnRecord).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete 删除代码生成业务（表信息 + 列信息）
// @function: Delete
// @description: 删除代码生成业务，同时清理列信息
// @param: ids []uint 表编号
// @return: err error
func (s Table) Delete(ids []uint) (err error) {
	if len(ids) == 0 {
		return errors.New("请选择要删除的数据")
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("table_id IN (?)", ids).Unscoped().Delete(&domain.TableColumn{}).Error; err != nil {
			return err
		}
		return tx.Where("table_id IN (?)", ids).Unscoped().Delete(&domain.Table{}).Error
	})
}

// physicalTable 获取表在数据库中的实际表名（含 gorm 表前缀配置）
// @function: physicalTable
// @description: 获取表在数据库中的实际表名
// @param: name string 逻辑表名
// @return: string
func physicalTable(name string) string {
	if database.DB == nil || database.DB.Config == nil || database.DB.Config.NamingStrategy == nil {
		return name
	}
	return database.DB.Config.NamingStrategy.TableName(name)
}
