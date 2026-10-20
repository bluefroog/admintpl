#!/bin/bash
# ─────────────────────────────────────────────────────────────
# admintpl 代码生成（app/gen）深度回归脚本
# 用途：对「已启动」的服务实例做一轮 代码生成 端到端校验：
#       建临时表 -> 导入表结构 -> 校验列配置推导 -> 修改数据库结构 -> 同步数据库 -> 校验 -> 清理
# 用法：./scripts/gen-check.sh [BASE_URL] [MYSQL_CMD]
#       MYSQL_CMD 形如：mysql -h127.0.0.1 -uroot -p123456 --batch --skip-column-names admintpl -e
#       默认值：docker exec mysql mysql -uroot -p123456 --batch --skip-column-names admintpl -e
# 依赖：bash、curl、mysql 客户端（脚本会自建自删临时表；不会修改业务表数据）
# ─────────────────────────────────────────────────────────────
set -u

BASE_URL="${1:-${BASE_URL:-http://127.0.0.1:8080}}"
MYSQL_CMD="${2:-${MYSQL_CMD:-docker exec mysql mysql -uroot -p123456 --batch --skip-column-names admintpl -e}}"
MYSQL() { $MYSQL_CMD "$1" 2>/dev/null; }

T="$BASE_URL/gen"
DEMO_TABLE="smoke_gen_demo"
ok=0; bad=0

# 登录获取令牌（与 smoke.sh 一致：认证关闭时自动降级为免认证）
TOKEN=$(curl -s -m 10 -X POST "$BASE_URL/login" -H 'Content-Type: application/json' \
  -d "{\"username\":\"${SMOKE_USER:-admin}\",\"password\":\"${SMOKE_PASS:-admin123}\"}" \
  | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
AUTH=(-H "Authorization: Bearer ${TOKEN}")

# chk "名称" "实际内容" "期望正则"
chk() {
  if echo "$2" | grep -qE "$3"; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-46s %s\n' "$1" "$(echo "$2" | head -c 120)"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-46s %s\n' "$1" "$(echo "$2" | head -c 200)"
  fi
}

# ---- 0. 清理历史残留 ----
MYSQL "DROP TABLE IF EXISTS $DEMO_TABLE;" >/dev/null
left=$(curl -s -m 10 "${AUTH[@]}" "$T/list?table_name=$DEMO_TABLE&pageSize=1")
if echo "$left" | grep -q '"total":1'; then
  lid=$(echo "$left" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
  curl -s -m 10 "${AUTH[@]}" -X DELETE "$T/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$lid]}" >/dev/null
fi

# ---- 1. 建临时表并导入 ----
MYSQL "CREATE TABLE $DEMO_TABLE (
  demo_id bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键',
  demo_name varchar(64) NOT NULL COMMENT '名称',
  content text NULL COMMENT '内容',
  price decimal(10,2) NULL COMMENT '金额',
  remark varchar(255) NULL COMMENT '备注',
  create_time datetime NULL DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (demo_id)) ENGINE=InnoDB COMMENT='冒烟代码生成表';" >/dev/null
out=$(curl -s -m 10 "${AUTH[@]}" -X POST "$T/import?tables=$DEMO_TABLE")
chk "导入表结构（tables=逗号分隔写法）" "$out" '"imported":1'

list=$(curl -s -m 10 "${AUTH[@]}" "$T/list?table_name=$DEMO_TABLE&pageSize=1")
gid=$(echo "$list" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
chk "类名（无前缀不去前缀）" "$list" '"class_name":"SmokeGenDemo"'
chk "业务名取表名最后一段" "$list" '"business_name":"demo"'
chk "功能名取表描述去掉「表」" "$list" '"function_name":"冒烟代码生成"'
chk "包路径/模块名/作者来自 system.gen 配置" "$list" '"package_name":"github.com/bluefroog/admintpl","module_name":"system"'
chk "作者来自 system.gen 配置" "$list" '"function_author":"bluefrog"'
chk "默认模板 crud" "$list" '"tpl_category":"crud"'

det=$(curl -s -m 10 "${AUTH[@]}" "$T/detail?id=$gid")
chk "主键列识别" "$det" "\"pk_column\":\{\"id\":[0-9]+,\"table_id\":[0-9]+,\"column_name\":\"demo_id\""
chk "主键列不参与编辑/列表/查询" "$det" "\"column_name\":\"demo_id\".*?\"is_edit\":\"0\",\"is_list\":\"0\",\"is_query\":\"0\""
chk "varchar(64) -> string/input，name 结尾必填且模糊查询" "$det" "\"column_name\":\"demo_name\".*?\"go_type\":\"string\",\"go_field\":\"DemoName\",\"html_field\":\"demoName\".*?\"is_required\":\"1\""
chk "text -> textarea" "$det" "\"column_name\":\"content\".*?\"html_type\":\"textarea\""
chk "decimal(10,2) -> float64" "$det" "\"column_name\":\"price\".*?\"go_type\":\"float64\""
chk "remark 不参与查询" "$det" "\"column_name\":\"remark\".*?\"is_query\":\"0\""
chk "create_time -> Time/datetime，不参与新增" "$det" "\"column_name\":\"create_time\".*?\"go_type\":\"Time\".*?\"is_insert\":\"0\""

# ---- 2. 同步数据库：新增列 + 删除列 + 类型变化 ----
MYSQL "ALTER TABLE $DEMO_TABLE ADD COLUMN demo_status char(1) NOT NULL DEFAULT '0' COMMENT '状态',
  MODIFY COLUMN demo_name varchar(128) NOT NULL COMMENT '名称',
  DROP COLUMN remark;" >/dev/null
out=$(curl -s -m 10 "${AUTH[@]}" -X POST "$T/synch-db" -H 'Content-Type: application/json' -d "{\"id\":$gid}")
chk "同步数据库表结构" "$out" '"code":0'
det=$(curl -s -m 10 "${AUTH[@]}" "$T/detail?id=$gid")
chk "同步后新增列已入库" "$det" '"column_name":"demo_status"'
chk "status 列 -> radio 且必填" "$det" "\"column_name\":\"demo_status\".*?\"is_required\":\"1\".*?\"html_type\":\"radio\""
chk "同步后已移除列被清理（remark 列不存在）" "$(echo "$det" | grep -c '"column_name":"remark"')" '^0$'
chk "列类型变化后 Go 类型重新推导" "$det" "\"column_name\":\"demo_name\".*?\"column_type\":\"varchar\(128\)\".*?\"go_type\":\"string\""

# ---- 3. 模板渲染：预览 / 下载 ----
pv=$(curl -s -m 10 "${AUTH[@]}" "$T/preview?id=$gid")
chk "预览返回 Go 实体代码" "$pv" 'package domain'
chk "预览类名取自表配置" "$pv" 'type SmokeGenDemo struct'
chk "表无 create_by 时不嵌入审计列" "$pv" '未嵌入 .core\.AuditModel.'
chk "预览返回业务逻辑" "$pv" 'func \(s SmokeGenDemo\) GetList\('
chk "预览返回控制器（含 swag 路由注解）" "$pv" '@Router /system/smoke_gen_demo/list'
chk "预览返回建表语句" "$pv" 'DROP TABLE IF EXISTS .smoke_gen_demo'
chk "预览返回接口测试用例" "$pv" 'Bearer \{\{token\}\}'
chk "预览返回接入说明" "$pv" '代码生成接入说明'
out=$(curl -s -m 10 "${AUTH[@]}" "$T/preview")
chk "预览缺少编号被拒绝" "$out" '"code":-1'
out=$(curl -s -m 10 "${AUTH[@]}" "$T/preview?id=999999")
chk "预览不存在的表被拒绝" "$out" '"code":-1'
zip_file=$(mktemp)
zip_meta=$(curl -s -m 20 "${AUTH[@]}" -o "$zip_file" -w '%{http_code} %{content_type}' "$T/download?id=$gid")
chk "下载 zip 返回 200 + application/zip" "$zip_meta" '^200 application/zip'
chk "下载内容是 zip（魔数 PK）" "$(head -c 2 "$zip_file")" '^PK$'
chk "zip 内含生成文件" "$(wc -c <"$zip_file" | tr -d ' ')" '^[1-9][0-9]*$'
rm -f "$zip_file"
out=$(curl -s -m 10 "${AUTH[@]}" "$T/download?id=999999")
chk "下载不存在的表被拒绝" "$out" '"code":-1'

# ---- 3.1 树表模板：tpl_category=tree 时按树编码升序 ----
out=$(curl -s -m 10 "${AUTH[@]}" -X PUT "$T/update" -H 'Content-Type: application/json' \
  -d "{\"id\":$gid,\"table_name\":\"$DEMO_TABLE\",\"class_name\":\"SmokeGenDemo\",\"tpl_category\":\"tree\",\"module_name\":\"system\",\"function_name\":\"冒烟\",\"tree_code\":\"demo_id\",\"tree_parent_code\":\"parent_id\",\"tree_name\":\"demo_name\"}")
chk "切换为树表模板" "$out" '"code":0'
pv=$(curl -s -m 10 "${AUTH[@]}" "$T/preview?id=$gid")
chk "树表列表按树编码升序" "$pv" 'db\.Order\(\\"demo_id asc\\"\)'
chk "树表接入说明输出树参数" "$pv" 'treeCode=demo_id.*treeParentCode=parent_id.*treeName=demo_name'

# ---- 4. 参数与过滤校验 ----
out=$(curl -s -m 10 "${AUTH[@]}" "$T/db-list?table_name=$DEMO_TABLE")
chk "已导入的表不再出现在 db-list" "$out" '"total":0'
out=$(curl -s -m 10 "${AUTH[@]}" "$T/detail?id=999999")
chk "不存在的表详情返回错误" "$out" '"code":-1'
out=$(curl -s -m 10 "${AUTH[@]}" -X POST "$T/import" -H 'Content-Type: application/json' -d '{"tables":["not_exist_table"]}')
chk "导入不存在的表被拒绝" "$out" '"code":-1'
out=$(curl -s -m 10 "${AUTH[@]}" -X POST "$T/synch-db" -H 'Content-Type: application/json' -d '{"id":999999}')
chk "同步不存在的表被拒绝" "$out" '"code":-1'

# ---- 5. 清理（接口删除 + 物理表删除） ----
out=$(curl -s -m 10 "${AUTH[@]}" -X DELETE "$T/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$gid]}")
chk "删除代码生成配置" "$out" '"code":0'
chk "字段配置已级联清理" "$(MYSQL "SELECT count(*) FROM gen_table_column WHERE table_id=$gid;")" '^0$'
chk "gen_table 无残留数据" "$(MYSQL "SELECT count(*) FROM gen_table;")" '^0$'
MYSQL "DROP TABLE IF EXISTS $DEMO_TABLE;" >/dev/null

printf '\033[36m===== gen 深度校验：通过=%s  失败=%s =====\033[0m\n' "$ok" "$bad"
[ "$bad" -eq 0 ] || exit 1
