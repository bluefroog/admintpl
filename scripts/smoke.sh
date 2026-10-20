#!/bin/bash
# ─────────────────────────────────────────────────────────────
# admintpl 接口冒烟测试脚本
# 用途：对「已启动」的服务实例做一轮 读接口 + 全模块 CRUD 回归，
#       用于容器/部署环境自检（尤其能发现 GORM 列名与表结构不匹配的问题）。
# 用法：./scripts/smoke.sh [BASE_URL]        默认 http://127.0.0.1:8080
#       just smoke [BASE_URL]
# 依赖：bash、curl（服务需已连接可写数据库；脚本会自建自删测试数据）
# ─────────────────────────────────────────────────────────────
set -u

BASE_URL="${1:-${BASE_URL:-http://127.0.0.1:8080}}"
B="$BASE_URL/system"
M="$BASE_URL/monitor"
S=$(date +%s)
ok=0; sqlbad=0; bad=0

# ---- 登录获取令牌（认证关闭或未初始化数据时自动降级为免认证模式） ----
SMOKE_USER="${SMOKE_USER:-admin}"
SMOKE_PASS="${SMOKE_PASS:-admin123}"
TOKEN=""
login_out=$(curl -s -m 10 -X POST "$BASE_URL/login" -H 'Content-Type: application/json' \
  -d "{\"username\":\"$SMOKE_USER\",\"password\":\"$SMOKE_PASS\"}")
TOKEN=$(echo "$login_out" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
# 始终携带认证头（令牌为空时等价于未认证；服务端认证关闭时该头会被忽略）
auth_header=(-H "Authorization: Bearer ${TOKEN}")
if [ -n "$TOKEN" ]; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s\n' "登录 /login" "已获取令牌（${#TOKEN} 位）"
else
  printf '\033[33mWARN\033[0m  %-44s %s\n' "登录 /login" "未获取到令牌，按免认证模式继续：$(echo "$login_out" | head -c 120)"
fi

# 取 JSON 中第一个 "id" 值（不依赖 jq）
pick_id() { echo "$1" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2; }

# 读接口断言：SQL 列名错误单独计数（这类错误一定是 bug）
req() {
  local m=$1 p=$2 out
  out=$(curl -s -m 10 "${auth_header[@]}" -X "$m" "$B$p")
  if echo "$out" | grep -q '"code":0'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "$p"
  elif echo "$out" | grep -qE '1054|Unknown column'; then
    sqlbad=$((sqlbad+1)); printf '\033[31mSQLERR\033[0m %-44s %s\n' "$p" "$(echo "$out" | head -c 170)"
  elif echo "$out" | grep -q 'record not found'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s （空表，无记录，符合预期）\n' "$p"
  else
    bad=$((bad+1)); printf '\033[33mWARN\033[0m  %-44s %s\n' "$p" "$(echo "$out" | head -c 170)"
  fi
}

# 监控模块读接口断言（与 req 相同，但使用 /monitor 前缀）
mreq() {
  local m=$1 p=$2 out
  out=$(curl -s -m 10 "${auth_header[@]}" -X "$m" "$M$p")
  if echo "$out" | grep -q '"code":0'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "monitor$p"
  elif echo "$out" | grep -qE '1054|Unknown column'; then
    sqlbad=$((sqlbad+1)); printf '\033[31mSQLERR\033[0m %-44s %s\n' "monitor$p" "$(echo "$out" | head -c 170)"
  elif echo "$out" | grep -q 'record not found'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s （空表，无记录，符合预期）\n' "monitor$p"
  else
    bad=$((bad+1)); printf '\033[33mWARN\033[0m  %-44s %s\n' "monitor$p" "$(echo "$out" | head -c 170)"
  fi
}

# 全模块 CRUD 断言：create → list(取 id) → detail → update → delete
crud_at() {
  local B=$1; shift
  local mod=$1 listq=$2 body=$3 ub=$4 out id
  out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$B/$mod/create" -H 'Content-Type: application/json' -d "$body")
  echo "$out" | grep -q '"code":0' || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  create %-10s %s\n' "$mod" "$out"; return; }
  out=$(curl -s -m 10 "${auth_header[@]}" "$B/$mod/list?pageSize=1&$listq")
  id=$(pick_id "$out")
  [ -n "$id" ] || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  list   %-10s [%s] %s\n' "$mod" "$listq" "$(echo "$out" | head -c 150)"; return; }
  out=$(curl -s -m 10 "${auth_header[@]}" "$B/$mod/detail?id=$id")
  echo "$out" | grep -q '"code":0' || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  detail %-10s id=%s %s\n' "$mod" "$id" "$out"; return; }
  out=$(curl -s -m 10 "${auth_header[@]}" -X PUT "$B/$mod/update" -H 'Content-Type: application/json' -d "$(echo "$ub" | sed "s/%ID%/$id/")")
  echo "$out" | grep -q '"code":0' || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  update %-10s id=%s %s\n' "$mod" "$id" "$out"; return; }
  out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/$mod/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$id]}")
  echo "$out" | grep -q '"code":0' || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  delete %-10s id=%s %s\n' "$mod" "$id" "$out"; return; }
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-10s CRUD 全通过 (id=%s)\n' "$mod" "$id"
}

crud() {
  crud_at "$B" "$@"
}

crudm() {
  crud_at "$M" "$@"
}

echo "===== 认证接口 ====="
authed() {
  local p=$1 out
  out=$(curl -s -m 10 "${auth_header[@]}" "$BASE_URL$p")
  if echo "$out" | grep -q '"code":0'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "$p"
  else
    bad=$((bad+1)); printf '\033[33mWARN\033[0m  %-44s %s\n' "$p" "$(echo "$out" | head -c 170)"
  fi
}
authed '/captchaImage'
authed '/getInfo'
authed '/getRouters'
if [ -n "$TOKEN" ]; then
  code=$(curl -s -m 10 -o /dev/null -w '%{http_code}' "$B/user/list")
  if [ "$code" = "401" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s 无令牌访问返回 401\n' '认证拦截'
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 期望 401，实际 %s\n' '认证拦截' "$code"
  fi
fi

echo
echo "===== 读接口（列表 / 详情 / 唯一性校验 / 树）====="
req GET '/config/list?pageSize=3'
req GET '/config/list?name=x&type=0&key=k'
req GET '/config/detail?id=1'
req GET '/config/check-key?key=x'
req GET '/config/get-by-key?key=x'

req GET '/department/list?pageSize=3'
req GET '/department/list?name=研&status=0'
req GET '/department/detail?id=1'
req GET '/department/check-name?name=x'
req GET '/department/tree'
req GET '/department/role-dept-tree?role_id=1'

req GET '/dict-data/list?pageSize=3'
req GET '/dict-data/list?type=x&label=x&status=0'
req GET '/dict-data/detail?id=1'
req GET '/dict-data/check-label?type=x&label=x'
req GET '/dict-data/get-by-type?type=x'

req GET '/dict-type/list?pageSize=3'
req GET '/dict-type/list?name=x&type=x&status=0'
req GET '/dict-type/detail?id=1'
req GET '/dict-type/check-type?type=x'
req GET '/dict-type/get-by-type?type=x'
req GET '/dict-type/list-all'

mreq GET '/job/list?pageSize=3'
mreq GET '/job/list?name=x&groupId=1&status=0'
mreq GET '/job/detail?id=1'
mreq GET '/job/check-name?name=x'
mreq GET '/jobLog/list?pageSize=3'
mreq GET '/jobLog/detail?id=1'

mreq GET '/logininfor/list?pageSize=3'
mreq GET '/logininfor/list?loginName=x&status=0'
mreq GET '/logininfor/detail?id=1'
mreq GET '/operlog/list?pageSize=3'
mreq GET '/operlog/list?title=x&businessType=1&status=0'
mreq GET '/operlog/detail?id=1'

req GET '/menu/list?pageSize=3'
req GET '/menu/list?name=x&status=0&menuType=M'
req GET '/menu/detail?id=1'
req GET '/menu/check-name?name=x'
req GET '/menu/check-perms?perms=x'
req GET '/menu/tree'
req GET '/menu/role-menu-tree?role_id=1'

req GET '/notice/list?pageSize=3'
req GET '/notice/detail?id=1'

req GET '/post/list?pageSize=3'
req GET '/post/list?name=x&code=x&status=0'
req GET '/post/detail?id=1'
req GET '/post/check-name?name=x'
req GET '/post/check-code?code=x'

req GET '/role/list?pageSize=3'
req GET '/role/list?name=x&key=x&status=0'
req GET '/role/detail?id=1'
req GET '/role/check-name?name=x'
req GET '/role/check-key?key=x'
req GET '/role/allocated-list?role_id=1'
req GET '/role/unallocated-list?role_id=1'

req GET '/role-dept/list?role_id=1'
req GET '/role-menu/list?role_id=1'
req GET '/user-post/list?user_id=1'
req GET '/user-role/list?user_id=1'

req GET '/user/list?pageSize=3'
req GET '/user/list?username=x&status=0&departmentId=1'
req GET '/user/detail?id=1'
req GET '/user/check-username?username=x'
req GET '/user/check-mobile?mobile=13800000000'
req GET '/user/check-email?email=x@x.com'
req GET '/user/posts?id=1'
req GET '/user/roles?id=1'

mreq GET '/online/list?pageSize=3'
mreq GET '/online/detail?id=1'

echo
echo "===== 写接口（全模块 CRUD 往返）====="
crud config     "name=冒烟参数$S"  "{\"name\":\"冒烟参数$S\",\"key\":\"smoke.key.$S\",\"value\":\"v1\",\"type\":\"N\",\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟参数$S改\",\"key\":\"smoke.key.$S\",\"value\":\"v2\",\"type\":\"N\",\"remark\":\"备注2\"}"
crud department "name=冒烟部门$S"  "{\"parent_id\":0,\"name\":\"冒烟部门$S\",\"ancestors\":\"0\",\"sort\":1,\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"parent_id\":0,\"name\":\"冒烟部门$S改\",\"ancestors\":\"0\",\"sort\":2,\"status\":0,\"remark\":\"备注2\"}"
crud dict-type  "name=冒烟字典$S"  "{\"name\":\"冒烟字典$S\",\"type\":\"smoke_type_$S\",\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟字典$S改\",\"type\":\"smoke_type_$S\",\"status\":0,\"remark\":\"备注2\"}"
crud dict-data  "type=smoke_type_$S" "{\"sort\":1,\"label\":\"冒烟标签$S\",\"value\":\"1\",\"type\":\"smoke_type_$S\",\"css_class\":\"\",\"list_class\":\"\",\"is_default\":\"0\",\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"sort\":2,\"label\":\"冒烟标签$S改\",\"value\":\"2\",\"type\":\"smoke_type_$S\",\"css_class\":\"\",\"list_class\":\"\",\"is_default\":\"0\",\"status\":0,\"remark\":\"备注2\"}"
crudm job       "name=冒烟任务$S"  "{\"name\":\"冒烟任务$S\",\"group\":\"default\",\"executor\":\"default\",\"invoke_target\":\"ryTask.ryNoParams()\",\"job_args\":\"\",\"job_kwargs\":\"\",\"cron_expression\":\"0 0 1 * * ?\",\"misfire_policy\":\"3\",\"concurrent\":\"0\",\"status\":1,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟任务$S改\",\"group\":\"default\",\"executor\":\"default\",\"invoke_target\":\"ryTask.ryParams('ry')\",\"job_args\":\"\",\"job_kwargs\":\"\",\"cron_expression\":\"0 0 1 * * ?\",\"misfire_policy\":\"3\",\"concurrent\":\"0\",\"status\":1,\"remark\":\"备注2\"}"
crud menu       "name=冒烟菜单$S"  "{\"name\":\"冒烟菜单$S\",\"parent_id\":0,\"sort\":1,\"path\":\"/smoke$S\",\"component\":\"\",\"query\":\"\",\"route_name\":\"\",\"is_frame\":1,\"is_cache\":0,\"menu_type\":\"M\",\"visible\":\"0\",\"status\":0,\"perms\":\"smoke:list$S\",\"icon\":\"#\"}"  "{\"id\":%ID%,\"name\":\"冒烟菜单$S改\",\"parent_id\":0,\"sort\":2,\"path\":\"/smoke$S\",\"component\":\"\",\"query\":\"\",\"route_name\":\"\",\"is_frame\":1,\"is_cache\":0,\"menu_type\":\"M\",\"visible\":\"0\",\"status\":0,\"perms\":\"smoke:list$S\",\"icon\":\"#\"}"
crud notice     "title=冒烟公告$S" "{\"title\":\"冒烟公告$S\",\"content\":\"公告内容\",\"type\":1,\"status\":0}"  "{\"id\":%ID%,\"title\":\"冒烟公告$S改\",\"content\":\"公告内容2\",\"type\":1,\"status\":0}"
crud post       "name=冒烟岗位$S"  "{\"code\":\"smoke_post_$S\",\"name\":\"冒烟岗位$S\",\"sort\":9,\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"code\":\"smoke_post_$S\",\"name\":\"冒烟岗位$S改\",\"sort\":8,\"status\":0,\"remark\":\"备注2\"}"
crud role       "name=冒烟角色$S"  "{\"key\":\"smoke_role_$S\",\"name\":\"冒烟角色$S\",\"sort\":9,\"status\":0,\"menu_check_strictly\":1,\"dept_check_strictly\":1,\"data_scope\":1,\"del_flag\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"key\":\"smoke_role_$S\",\"name\":\"冒烟角色$S改\",\"sort\":8,\"status\":0,\"menu_check_strictly\":1,\"dept_check_strictly\":1,\"data_scope\":1,\"del_flag\":0,\"remark\":\"备注2\"}"
crud user       "username=smoke$S" "{\"username\":\"smoke$S\",\"mobile\":\"139$S\",\"nickname\":\"冒烟用户$S\",\"status\":1,\"email\":\"smoke$S@test.com\",\"gender\":1,\"avatar\":\"\",\"departmentID\":0,\"remark\":\"备注1\",\"isAdmin\":false}"  "{\"id\":%ID%,\"username\":\"smoke$S\",\"mobile\":\"139$S\",\"nickname\":\"冒烟用户$S改\",\"status\":1,\"email\":\"smoke$S@test.com\",\"gender\":1,\"avatar\":\"\",\"departmentID\":0,\"remark\":\"备注2\",\"isAdmin\":false}"

echo
echo "===== 定时任务调度（cron 解析 / 自动触发 / 立即执行 / 启停 / 日志）====="
JOBN="smokejob$S"
# 非法 cron 表达式应当被拒绝
out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$M/job/create" -H 'Content-Type: application/json' \
  -d "{\"name\":\"${JOBN}bad\",\"invoke_target\":\"ryTask.ryNoParams()\",\"cron_expression\":\"0 0 0 * *\",\"status\":1}")
if echo "$out" | grep -q 'Cron 表达式不正确'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '非法 cron 表达式被拒绝'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '非法 cron 表达式被拒绝' "$(echo "$out" | head -c 170)"
fi
# 未注册的调用目标应当被拒绝
out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$M/job/create" -H 'Content-Type: application/json' \
  -d "{\"name\":\"${JOBN}bad2\",\"invoke_target\":\"evil.exec('rm -rf /')\",\"cron_expression\":\"0 0 1 * * ?\",\"status\":1}")
if echo "$out" | grep -q '调用目标字符串错误'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '未注册的调用目标被拒绝'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '未注册的调用目标被拒绝' "$(echo "$out" | head -c 170)"
fi

# 任务日志条数（按任务名称统计）
job_logs() {
  curl -s -m 10 "${auth_header[@]}" "$M/jobLog/list?pageSize=200&job_name=$1" | grep -o '"job_name"' | wc -l | tr -d ' '
}
# 每 2 秒执行一次、禁止并发、错过执行放弃
out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$M/job/create" -H 'Content-Type: application/json' \
  -d "{\"name\":\"$JOBN\",\"group\":\"default\",\"executor\":\"default\",\"invoke_target\":\"ryTask.ryNoParams()\",\"cron_expression\":\"*/2 * * * * ?\",\"misfire_policy\":\"3\",\"concurrent\":\"1\",\"status\":0}")
if echo "$out" | grep -q '"code":0'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '创建并启用定时任务'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '创建并启用定时任务' "$(echo "$out" | head -c 170)"
fi
JID=$(pick_id "$(curl -s -m 10 "${auth_header[@]}" "$M/job/list?pageSize=1&name=$JOBN")")
if [ -n "$JID" ]; then
  sleep 6
  c1=$(job_logs "$JOBN")
  if [ "${c1:-0}" -ge 1 ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s 已产生 %s 条执行日志\n' '任务按 cron 自动触发' "$c1"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 未产生执行日志\n' '任务按 cron 自动触发'
  fi
  # 暂停后不应再自动触发
  out=$(curl -s -m 10 "${auth_header[@]}" -X PUT "$M/job/change-status" -H 'Content-Type: application/json' -d "{\"id\":$JID,\"status\":1}")
  echo "$out" | grep -q '"code":0' && { ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '暂停定时任务'; } || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '暂停定时任务' "$out"; }
  sleep 6
  c2=$(job_logs "$JOBN")
  if [ "${c2:-0}" = "${c1:-0}" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s 日志条数保持 %s\n' '暂停后不再自动触发' "$c2"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 暂停前 %s 条、暂停后 %s 条\n' '暂停后不再自动触发' "$c1" "$c2"
  fi
  # 立即执行一次（暂停状态同样可执行）
  out=$(curl -s -m 10 "${auth_header[@]}" -X PUT "$M/job/run" -H 'Content-Type: application/json' -d "{\"id\":$JID}")
  echo "$out" | grep -q '"code":0' && { ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '立即执行一次任务'; } || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '立即执行一次任务' "$out"; }
  sleep 2
  c3=$(job_logs "$JOBN")
  if [ "${c3:-0}" -gt "${c2:-0}" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s -> %s 条\n' '执行日志写入 sys_job_log' "$c2" "$c3"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 日志未增加（%s）\n' '执行日志写入 sys_job_log' "$c3"
  fi
  # 删除任务应同时清理其执行日志（与若依一致）
  out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$M/job/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$JID]}")
  echo "$out" | grep -q '"code":0' && { ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '删除定时任务'; } || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '删除定时任务' "$out"; }
  c4=$(job_logs "$JOBN")
  if [ "${c4:-1}" = "0" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '删除任务同时清理执行日志'
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 仍残留 %s 条日志\n' '删除任务同时清理执行日志' "$c4"
  fi
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 未取到任务ID\n' '定时任务调度'
fi

echo
echo "===== 数据权限（角色数据范围：本部门 / 本部门及以下 / 仅本人 / 自定义）====="
# 构造：部门A（父）→部门A2（子）、部门B（兄弟），一个「本部门」数据范围的角色，
#       两个分属部门A/部门B的用户，并以部门A的用户登录验证数据范围过滤
DSA="dsa$S"; DSB="dsb$S"; DSROL="dsrole$S"
post() { curl -s -m 10 "${auth_header[@]}" -X POST "$B$1" -H 'Content-Type: application/json' -d "$2"; }
put()  { curl -s -m 10 "${auth_header[@]}" -X PUT "$B$1" -H 'Content-Type: application/json' -d "$2"; }
del()  { curl -s -m 10 "${auth_header[@]}" -X DELETE "$B$1" -H 'Content-Type: application/json' -d "$2"; }
getq() { curl -s -m 10 "${auth_header[@]}" "$B$1"; }
post /department/create "{\"parent_id\":0,\"name\":\"数据权限A$S\",\"ancestors\":\"0\",\"sort\":9,\"status\":0}" >/dev/null
DEPA=$(pick_id "$(getq "/department/list?pageSize=1&name=数据权限A$S")")
post /department/create "{\"parent_id\":$DEPA,\"name\":\"数据权限A2$S\",\"ancestors\":\"0,$DEPA\",\"sort\":9,\"status\":0}" >/dev/null
DEPA2=$(pick_id "$(getq "/department/list?pageSize=1&name=数据权限A2$S")")
post /department/create "{\"parent_id\":0,\"name\":\"数据权限B$S\",\"ancestors\":\"0\",\"sort\":9,\"status\":0}" >/dev/null
DEPB=$(pick_id "$(getq "/department/list?pageSize=1&name=数据权限B$S")")
post /role/create "{\"key\":\"$DSROL\",\"name\":\"数据权限角色$S\",\"sort\":9,\"status\":0,\"menu_check_strictly\":1,\"dept_check_strictly\":1,\"data_scope\":3,\"del_flag\":0}" >/dev/null
ROLD=$(pick_id "$(getq "/role/list?pageSize=1&name=数据权限角色$S")")
post /user/create "{\"username\":\"$DSA\",\"mobile\":\"137$S\",\"nickname\":\"数据权限用户A$S\",\"status\":1,\"email\":\"$DSA@test.com\",\"gender\":1,\"departmentID\":$DEPA,\"remark\":\"数据权限\"}" >/dev/null
USERA=$(pick_id "$(getq "/user/list?pageSize=1&username=$DSA")")
post /user/create "{\"username\":\"$DSB\",\"mobile\":\"136$S\",\"nickname\":\"数据权限用户B$S\",\"status\":1,\"email\":\"$DSB@test.com\",\"gender\":1,\"departmentID\":$DEPB,\"remark\":\"数据权限\"}" >/dev/null
USERB=$(pick_id "$(getq "/user/list?pageSize=1&username=$DSB")")
put /user/reset-pwd "{\"id\":$USERA,\"password\":\"admin123\"}" >/dev/null
put /user/roles "{\"id\":$USERA,\"role_ids\":[$ROLD]}" >/dev/null
TOKENA=$(curl -s -m 10 -X POST "$BASE_URL/login" -H 'Content-Type: application/json' \
  -d "{\"username\":\"$DSA\",\"password\":\"admin123\"}" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$DEPA" ] && [ -n "$DEPA2" ] && [ -n "$DEPB" ] && [ -n "$ROLD" ] && [ -n "$USERA" ] && [ -n "$USERB" ] && [ -n "$TOKENA" ]; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s 角色=%s 部门=%s/%s/%s 用户=%s/%s\n' '构造数据权限测试数据' "$ROLD" "$DEPA" "$DEPA2" "$DEPB" "$USERA" "$USERB"
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 角色=%s 部门=%s/%s/%s 用户=%s/%s 子账号令牌=%s\n' '构造数据权限测试数据' "$ROLD" "$DEPA" "$DEPA2" "$DEPB" "$USERA" "$USERB" "${TOKENA:0:8}"
fi

# 数据权限断言：以子账号令牌查询，校验响应中「应包含 / 不应包含」关键字
ds_assert() {
  local want=$1 keyword=$2 desc=$3 path=$4 out
  out=$(curl -s -m 10 -H "Authorization: Bearer $TOKENA" "$B$path")
  if ! echo "$out" | grep -q '"code":0'; then
    if echo "$out" | grep -qE '1054|Unknown column'; then sqlbad=$((sqlbad+1)); else bad=$((bad+1)); fi
    printf '\033[31mFAIL\033[0m  %-44s 请求失败：%s\n' "$desc" "$(echo "$out" | head -c 170)"
    return
  fi
  if echo "$out" | grep -q "$keyword"; then
    if [ "$want" = in ]; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "$desc"
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 不应出现 %s\n' "$desc" "$keyword"
    fi
  else
    if [ "$want" = out ]; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "$desc"
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s 未找到 %s：%s\n' "$desc" "$keyword" "$(echo "$out" | head -c 150)"
    fi
  fi
}
if [ -n "$TOKENA" ]; then
  # 本部门数据权限（data_scope=3）
  ds_assert in  "$DSA"           '数据权限-本部门：用户列表含本部门用户'   '/user/list?pageSize=200'
  ds_assert out "$DSB"           '数据权限-本部门：用户列表不含他部门用户' '/user/list?pageSize=200'
  ds_assert in  "数据权限A$S"     '数据权限-本部门：部门列表含本部门'     '/department/list?pageSize=200'
  ds_assert out "数据权限A2$S"    '数据权限-本部门：部门列表不含子部门'   '/department/list?pageSize=200'
  ds_assert out "数据权限B$S"     '数据权限-本部门：部门列表不含他部门'   '/department/list?pageSize=200'
  ds_assert in  "数据权限角色$S"   '数据权限-角色列表：含本部门用户所属角色' '/role/list?pageSize=200'
  # 本部门及以下数据权限（data_scope=4）
  put /role/data-scope "{\"id\":$ROLD,\"data_scope\":4,\"dept_ids\":[]}" >/dev/null
  ds_assert in  "数据权限A2$S"    '数据权限-本部门及以下：含子部门'       '/department/list?pageSize=200'
  ds_assert out "数据权限B$S"     '数据权限-本部门及以下：不含他部门'     '/department/list?pageSize=200'
  ds_assert in  "$DSA"           '数据权限-本部门及以下：用户列表含本部门用户' '/user/list?pageSize=200'
  # 仅本人数据权限（data_scope=5）：部门表无用户ID列，该条件恒不成立
  put /role/data-scope "{\"id\":$ROLD,\"data_scope\":5}" >/dev/null
  ds_assert in  "$DSA"           '数据权限-仅本人：用户列表含本人'       '/user/list?pageSize=200'
  ds_assert out "$DSB"           '数据权限-仅本人：用户列表不含他人'     '/user/list?pageSize=200'
  ds_assert out "数据权限A$S"     '数据权限-仅本人：部门列表为空（无用户列）' '/department/list?pageSize=200'
  # 自定义数据权限（data_scope=2，勾选部门B）
  put /role/data-scope "{\"id\":$ROLD,\"data_scope\":2,\"dept_ids\":[$DEPB]}" >/dev/null
  ds_assert in  "数据权限B$S"     '数据权限-自定义：部门列表含勾选部门'   '/department/list?pageSize=200'
  ds_assert out "数据权限A$S"     '数据权限-自定义：部门列表不含未勾选部门' '/department/list?pageSize=200'
  ds_assert in  "$DSB"           '数据权限-自定义：用户列表含勾选部门用户' '/user/list?pageSize=200'
  ds_assert out "$DSA"           '数据权限-自定义：用户列表不含未勾选部门用户' '/user/list?pageSize=200'
fi
# 非法数据范围必须被拒绝（1全部 2自定义 3本部门 4本部门及以下 5仅本人）
out=$(post /role/create "{\"key\":\"${DSROL}bad\",\"name\":\"非法数据范围角色$S\",\"data_scope\":9}")
if echo "$out" | grep -q '数据范围不正确'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '非法数据范围被拒绝（新增角色）'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '非法数据范围被拒绝（新增角色）' "$(echo "$out" | head -c 170)"
fi
out=$(put /role/data-scope "{\"id\":$ROLD,\"data_scope\":9}")
if echo "$out" | grep -q '数据范围不正确'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '非法数据范围被拒绝（修改数据权限）'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '非法数据范围被拒绝（修改数据权限）' "$(echo "$out" | head -c 170)"
fi

# 关联数据清理校验：先建立 用户-角色 / 用户-岗位 / 角色-菜单 / 角色-部门 四类关联
put /role/menu "{\"id\":$ROLD,\"menu_ids\":[1]}" >/dev/null
put /user/posts "{\"id\":$USERA,\"post_ids\":[1]}" >/dev/null
rel_total() { getq "$1" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2; }
rel_assert() {
  local want=$1 path=$2 desc=$3 got
  got=$(rel_total "$path")
  if [ "$got" = "$want" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-46s 关联记录数=%s\n' "$desc" "$got"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-46s 期望 %s 实际 %s\n' "$desc" "$want" "${got:-空}"
  fi
}
rel_assert 1 "/user-role/list?user_id=$USERA" '删除用户前：用户-角色关联存在'
rel_assert 1 "/user-post/list?user_id=$USERA" '删除用户前：用户-岗位关联存在'
rel_assert 1 "/role-menu/list?role_id=$ROLD"  '删除角色前：角色-菜单关联存在'
rel_assert 1 "/role-dept/list?role_id=$ROLD"  '删除角色前：角色-部门关联存在'

# 清理（静默）：解绑角色 → 退出登录 → 删除用户 / 角色 / 部门（顺序不可颠倒）
put /user/roles "{\"id\":$USERA,\"role_ids\":[]}" >/dev/null
curl -s -m 10 -X POST "$BASE_URL/logout" -H "Authorization: Bearer $TOKENA" >/dev/null
[ -n "$USERA" ] && del /user/delete "{\"ids\":[$USERA]}" >/dev/null
[ -n "$USERB" ] && del /user/delete "{\"ids\":[$USERB]}" >/dev/null
[ -n "$ROLD" ] && del /role/delete "{\"ids\":[$ROLD]}" >/dev/null
[ -n "$DEPA2" ] && del /department/delete "{\"ids\":[$DEPA2]}" >/dev/null
[ -n "$DEPA" ] && del /department/delete "{\"ids\":[$DEPA]}" >/dev/null
[ -n "$DEPB" ] && del /department/delete "{\"ids\":[$DEPB]}" >/dev/null

# 删除用户/角色必须同步清理关联表，避免遗留孤儿关联数据（与若依一致）
rel_assert 0 "/user-role/list?user_id=$USERA" '删除用户后：用户-角色关联已清理'
rel_assert 0 "/user-post/list?user_id=$USERA" '删除用户后：用户-岗位关联已清理'
rel_assert 0 "/role-menu/list?role_id=$ROLD"  '删除角色后：角色-菜单关联已清理'
rel_assert 0 "/role-dept/list?role_id=$ROLD"  '删除角色后：角色-部门关联已清理'

echo
echo "===== 字段零值落库校验（GORM default 标签陷阱）====="
# 带 default 标签的字段（如 menu.is_frame 默认 1、user.status 默认 1）用结构体 Create 时，
# 显式传入的零值会被数据库默认值覆盖，这里逐项校验零值能原样落库。
zero_check() {
  local mod=$1 listq=$2 body=$3 key=$4 out id
  out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$B/$mod/create" -H 'Content-Type: application/json' -d "$body")
  echo "$out" | grep -q '"code":0' || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  create %-10s %s\n' "$mod" "$out"; return; }
  out=$(curl -s -m 10 "${auth_header[@]}" "$B/$mod/list?pageSize=1&$listq")
  id=$(pick_id "$out")
  out=$(curl -s -m 10 "${auth_header[@]}" "$B/$mod/detail?id=$id")
  if echo "$out" | grep -q "\"$key\":0"; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-10s 零值字段 %s=0 落库正确 (id=%s)\n' "$mod" "$key" "$id"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-10s 零值字段 %s 被默认值覆盖 (id=%s) %s\n' "$mod" "$key" "$id" "$(echo "$out" | head -c 150)"
  fi
  curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/$mod/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$id]}" >/dev/null
}
zero_check menu "name=零值菜单$S" "{\"name\":\"零值菜单$S\",\"parent_id\":0,\"sort\":1,\"path\":\"/zero$S\",\"menu_type\":\"M\",\"is_frame\":0,\"is_cache\":1,\"visible\":\"0\",\"status\":0,\"perms\":\"zero:list$S\",\"icon\":\"#\"}" is_frame
zero_check role "name=零值角色$S" "{\"key\":\"zero_role_$S\",\"name\":\"零值角色$S\",\"sort\":9,\"status\":0,\"menu_check_strictly\":0,\"dept_check_strictly\":0,\"data_scope\":1,\"del_flag\":0,\"remark\":\"零值校验\"}" menu_check_strictly
zero_check user "username=zero$S" "{\"username\":\"zero$S\",\"mobile\":\"138$S\",\"nickname\":\"零值用户$S\",\"status\":0,\"email\":\"zero$S@test.com\",\"gender\":0,\"avatar\":\"\",\"departmentID\":0,\"remark\":\"零值校验\",\"isAdmin\":false}" status

echo
cache_names=$(curl -s -m 10 "${auth_header[@]}" "$M/cache")
echo "===== 参数 / 字典缓存（若依缓存语义：预热 / 失效 / 刷新）====="
# 取当前参数值（同时触发缓存写入）
cfg=$(curl -s -m 10 "${auth_header[@]}" "$B/config/get-by-key?key=sys.index.sideTheme")
cfg_value=$(echo "$cfg" | grep -o '"data":"[^"]*"' | head -1 | cut -d'"' -f4)
if echo "$cfg" | grep -q '"code":0'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s 当前值=%s\n' '读取参数 sys.index.sideTheme' "$cfg_value"
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '读取参数 sys.index.sideTheme' "$(echo "$cfg" | head -c 170)"
fi
# 参数已缓存：缓存列表按名称过滤应能查到该键
cl=$(curl -s -m 10 "${auth_header[@]}" "$M/cache/list?cacheName=sys_config&cacheKey=sys.index.sideTheme&pageSize=10")
if echo "$cl" | grep -q 'sys.index.sideTheme'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '参数已写入缓存 sys_config'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '参数已写入缓存 sys_config' "$(echo "$cl" | head -c 170)"
fi
# 修改参数后缓存应失效：再次按键取值必须拿到新值
cfg_list=$(curl -s -m 10 "${auth_header[@]}" "$B/config/list?key=sys.index.sideTheme&pageSize=1")
cfg_id=$(pick_id "$cfg_list")
new_value="theme-smoke$S"
upd=$(curl -s -m 10 "${auth_header[@]}" -X PUT "$B/config/update" -H 'Content-Type: application/json' \
  -d "{\"id\":$cfg_id,\"name\":\"主框架页-侧边栏主题\",\"key\":\"sys.index.sideTheme\",\"type\":\"Y\",\"value\":\"$new_value\",\"remark\":\"冒烟测试\"}")
after=$(curl -s -m 10 "${auth_header[@]}" "$B/config/get-by-key?key=sys.index.sideTheme")
if echo "$upd" | grep -q '"code":0' && echo "$after" | grep -q "$new_value"; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '修改参数后缓存失效并回源新值'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s upd=%s after=%s\n' '修改参数后缓存失效并回源新值' "$(echo "$upd" | head -c 90)" "$(echo "$after" | head -c 90)"
fi
# 还原参数值
if [ "$cfg_value" != "$new_value" ] && [ -n "$cfg_id" ]; then
  curl -s -m 10 "${auth_header[@]}" -X PUT "$B/config/update" -H 'Content-Type: application/json' \
    -d "{\"id\":$cfg_id,\"name\":\"主框架页-侧边栏主题\",\"key\":\"sys.index.sideTheme\",\"type\":\"Y\",\"value\":\"$cfg_value\",\"remark\":\"系统内置参数\"}" >/dev/null
fi
# 刷新参数缓存
out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/config/refresh-cache")
if echo "$out" | grep -qE '"code":0,"data":[1-9]'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s\n' '刷新参数缓存 /system/config/refresh-cache' "$(echo "$out" | grep -o '"data":[0-9]*')"
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '刷新参数缓存 /system/config/refresh-cache' "$(echo "$out" | head -c 170)"
fi
# 字典数据按类型读取后应写入字典缓存
dd=$(curl -s -m 10 "${auth_header[@]}" "$B/dict-data/get-by-type?type=sys_normal_disable")
dl=$(curl -s -m 10 "${auth_header[@]}" "$M/cache/list?cacheName=sys_dict&cacheKey=sys_normal_disable&pageSize=10")
if echo "$dd" | grep -q '"code":0' && echo "$dl" | grep -q 'sys_normal_disable'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '字典数据已写入缓存 sys_dict'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s dd=%s dl=%s\n' '字典数据已写入缓存 sys_dict' "$(echo "$dd" | head -c 90)" "$(echo "$dl" | head -c 90)"
fi
# 刷新字典缓存
out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/dict-type/refresh-cache")
if echo "$out" | grep -qE '"code":0,"data":[1-9]'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s\n' '刷新字典缓存 /system/dict-type/refresh-cache' "$(echo "$out" | grep -o '"data":[0-9]*')"
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '刷新字典缓存 /system/dict-type/refresh-cache' "$(echo "$out" | head -c 170)"
fi
# 字典类型变更后应清理该类型的字典缓存
dtype="smoke_dict_$S"
ct=$(curl -s -m 10 "${auth_header[@]}" -X POST "$B/dict-type/create" -H 'Content-Type: application/json' \
  -d "{\"name\":\"冒烟字典$S\",\"type\":\"$dtype\",\"status\":0,\"remark\":\"冒烟测试\"}")
dt_list=$(curl -s -m 10 "${auth_header[@]}" "$B/dict-type/list?type=$dtype&pageSize=1")
dt_id=$(pick_id "$dt_list")
cd_out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$B/dict-data/create" -H 'Content-Type: application/json' \
  -d "{\"label\":\"冒烟标签\",\"value\":\"1\",\"type\":\"$dtype\",\"sort\":1,\"is_default\":\"N\",\"status\":0,\"remark\":\"冒烟测试\"}")
cd_list=$(curl -s -m 10 "${auth_header[@]}" "$B/dict-data/list?type=$dtype&pageSize=1")
dd_id=$(pick_id "$cd_list")
curl -s -m 10 "${auth_header[@]}" "$B/dict-data/get-by-type?type=$dtype" >/dev/null
dl2=$(curl -s -m 10 "${auth_header[@]}" "$M/cache/list?cacheName=sys_dict&cacheKey=$dtype&pageSize=10")
del_t=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/dict-type/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$dt_id]}")
dl3=$(curl -s -m 10 "${auth_header[@]}" "$M/cache/list?cacheName=sys_dict&cacheKey=$dtype&pageSize=10")
if echo "$dl2" | grep -q "$dtype" && echo "$del_t" | grep -q '"code":0' && ! echo "$dl3" | grep -q "$dtype"; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '删除字典类型后清理字典缓存'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s cached=%s del=%s after=%s\n' '删除字典类型后清理字典缓存' "$(echo "$dl2" | head -c 60)" "$(echo "$del_t" | head -c 60)" "$(echo "$dl3" | head -c 60)"
fi
# 清理冒烟字典数据
[ -n "$dd_id" ] && curl -s -m 10 "${auth_header[@]}" -X DELETE "$B/dict-data/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$dd_id]}" >/dev/null

# 两个业务缓存已注册到缓存监控
if echo "$cache_names" | grep -q '"sys_config"' && echo "$cache_names" | grep -q '"sys_dict"'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '参数/字典缓存已注册到缓存监控'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '参数/字典缓存已注册到缓存监控' "$(echo "$cache_names" | head -c 170)"
fi

echo
echo "===== 服务监控 / 缓存监控 ====="
# 服务监控：CPU / 内存 / Go 运行时 / 主机 / 磁盘
srv=$(curl -s -m 10 "${auth_header[@]}" "$M/server")
srv_assert() {
  local key=$1 name=$2
  if echo "$srv" | grep -q "\"$key\""; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "$name"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' "$name" "$(echo "$srv" | head -c 170)"
  fi
}
srv_assert cpuNum            '服务监控返回 CPU 核心数'
srv_assert memTotal          '服务监控返回内存信息'
srv_assert goVersion         '服务监控返回 Go 运行时信息'
srv_assert sysComputerName   '服务监控返回服务器名称/IP'
srv_assert sysFiles          '服务监控返回磁盘信息'
if echo "$srv" | grep -qE '"cpuNum":[1-9]'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '服务监控 CPU 核心数 >= 1'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '服务监控 CPU 核心数 >= 1' "$(echo "$srv" | head -c 170)"
fi

# 缓存监控 / 缓存列表
mreq GET '/cache'
mreq GET '/cache/list?pageSize=5'
cache_info=$(curl -s -m 10 "${auth_header[@]}" "$M/cache")
if echo "$cache_info" | grep -q '"commandStats"'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '缓存监控返回读写命令统计'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '缓存监控返回读写命令统计' "$(echo "$cache_info" | head -c 170)"
fi
if echo "$cache_info" | grep -q '"captcha"'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '验证码缓存已纳入缓存监控'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '验证码缓存已纳入缓存监控' "$(echo "$cache_info" | head -c 170)"
fi
# 清空指定缓存与全部缓存
out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$M/cache/clear?cacheName=captcha")
echo "$out" | grep -q '"code":0' && { ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '清空指定缓存'; } || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '清空指定缓存' "$out"; }
out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$M/cache/clear")
echo "$out" | grep -q '"code":0' && { ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '清空全部缓存'; } || { bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '清空全部缓存' "$out"; }
out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$M/cache/clear?cacheName=not-exist")
if echo "$out" | grep -q '"code":-1'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '不存在的缓存名称被拒绝'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '不存在的缓存名称被拒绝' "$(echo "$out" | head -c 170)"
fi

echo
echo "===== 代码生成（表结构导入 / 配置修改 / 同步数据库）====="
T="$BASE_URL/tool"

# 代码生成接口断言（与 req 相同，但使用 /tool 前缀）
treq() {
  local m=$1 p=$2 out
  out=$(curl -s -m 10 "${auth_header[@]}" -X "$m" "$T$p")
  if echo "$out" | grep -q '"code":0'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' "tool$p"
  elif echo "$out" | grep -qE '1054|Unknown column'; then
    sqlbad=$((sqlbad+1)); printf '\033[31mSQLERR\033[0m %-44s %s\n' "tool$p" "$(echo "$out" | head -c 170)"
  elif echo "$out" | grep -q 'record not found'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s （空表，无记录，符合预期）\n' "tool$p"
  else
    bad=$((bad+1)); printf '\033[33mWARN\033[0m  %-44s %s\n' "tool$p" "$(echo "$out" | head -c 170)"
  fi
}

treq GET '/gen/list?pageSize=3'

# 未导入的数据库表列表（取第一张表用于导入测试）
db_list=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/db-list?pageSize=5")
imp_table=$(echo "$db_list" | grep -o '"table_name":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$imp_table" ]; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s\n' '数据库表列表返回可导入表' "$imp_table"
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '数据库表列表返回可导入表' "$(echo "$db_list" | head -c 170)"
fi

# 空表名导入被拒绝（参数校验）
out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$T/gen/import" -H 'Content-Type: application/json' -d '{"tables":[]}')
if echo "$out" | grep -q '"code":-1'; then
  ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '导入表结构（空表名被拒绝）'
else
  bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '导入表结构（空表名被拒绝）' "$(echo "$out" | head -c 170)"
fi

if [ -n "$imp_table" ]; then
  # 导入表结构
  out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$T/gen/import" -H 'Content-Type: application/json' -d "{\"tables\":[\"$imp_table\"]}")
  if echo "$out" | grep -q '"imported":1'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s\n' '导入表结构' "$imp_table"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '导入表结构' "$(echo "$out" | head -c 170)"
  fi

  # 重复导入被拒绝
  out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$T/gen/import" -H 'Content-Type: application/json' -d "{\"tables\":[\"$imp_table\"]}")
  if echo "$out" | grep -q '"code":-1'; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '重复导入同表被拒绝'
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '重复导入同表被拒绝' "$(echo "$out" | head -c 170)"
  fi

  # 导入后按表名查询取编号
  gen_list=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/list?table_name=$imp_table&pageSize=1")
  gen_id=$(pick_id "$gen_list")
  if [ -n "$gen_id" ]; then
    ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s id=%s\n' '代码生成表列表查询' "$gen_id"
  else
    bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '代码生成表列表查询' "$(echo "$gen_list" | head -c 170)"
  fi

  if [ -n "$gen_id" ]; then
    # 详情：列信息 + 主键列 + 类名（sys_ 前缀已按配置去除）
    gen_detail=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/detail?id=$gen_id")
    if echo "$gen_detail" | grep -q '"columns"' && echo "$gen_detail" | grep -q '"pk_column"'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '代码生成表详情返回列与主键列'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '代码生成表详情返回列与主键列' "$(echo "$gen_detail" | head -c 170)"
    fi
    if echo "$gen_detail" | grep -q '"go_type"'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '列信息包含推导的 GO 类型'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '列信息包含推导的 GO 类型' "$(echo "$gen_detail" | head -c 170)"
    fi

    # 字段列表接口
    treq GET "/gen/column-list?table_id=$gen_id"
    # 缺少表编号时被拒绝
    out=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/column-list")
    if echo "$out" | grep -q '"code":-1'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '字段列表缺少表编号被拒绝'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '字段列表缺少表编号被拒绝' "$(echo "$out" | head -c 170)"
    fi

    # 修改生成配置
    out=$(curl -s -m 10 "${auth_header[@]}" -X PUT "$T/gen/update" -H 'Content-Type: application/json' \
      -d "{\"id\":$gen_id,\"table_name\":\"$imp_table\",\"table_comment\":\"冒烟测试表\",\"class_name\":\"SmokeGen\",\"tpl_category\":\"crud\",\"package_name\":\"github.com/bluefroog/admintpl\",\"module_name\":\"system\",\"business_name\":\"smoke\",\"function_name\":\"冒烟测试\",\"function_author\":\"smoke\",\"gen_type\":\"0\",\"tree_code\":\"id\",\"remark\":\"smoke\"}")
    if echo "$out" | grep -q '"code":0'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '修改代码生成配置'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '修改代码生成配置' "$(echo "$out" | head -c 170)"
    fi
    # 修改结果与树表参数落库校验
    gen_detail=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/detail?id=$gen_id")
    if echo "$gen_detail" | grep -q '"class_name":"SmokeGen"' && echo "$gen_detail" | grep -q '"tree_code":"id"'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '配置修改与树表参数已落库'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '配置修改与树表参数已落库' "$(echo "$gen_detail" | head -c 170)"
    fi

    # 同步数据库表结构
    out=$(curl -s -m 10 "${auth_header[@]}" -X POST "$T/gen/synch-db" -H 'Content-Type: application/json' -d "{\"id\":$gen_id}")
    if echo "$out" | grep -q '"code":0'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '同步数据库表结构'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '同步数据库表结构' "$(echo "$out" | head -c 170)"
    fi

    # 预览生成代码（配置中 class_name/function_name 已改成 SmokeGen / 冒烟测试）
    out=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/preview?id=$gen_id")
    if echo "$out" | grep -q '"code":0' && echo "$out" | grep -q 'package domain' \
      && echo "$out" | grep -q 'type SmokeGen struct' && echo "$out" | grep -q '@Router /system/'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '预览生成代码（实体/控制器/路由注解）'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '预览生成代码（实体/控制器/路由注解）' "$(echo "$out" | head -c 170)"
    fi
    # 预览内容包含业务逻辑与建表语句、接入说明
    if echo "$out" | grep -q 'func (s SmokeGen) GetList(' && echo "$out" | grep -q 'DROP TABLE IF EXISTS' \
      && echo "$out" | grep -q '代码生成接入说明' && echo "$out" | grep -q 'Bearer {{token}}'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '预览生成代码（service/SQL/http/接入说明）'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '预览生成代码（service/SQL/http/接入说明）' "$(echo "$out" | head -c 170)"
    fi
    # 预览缺少编号被拒绝
    out=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/preview")
    if echo "$out" | grep -q '"code":-1'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '预览生成代码缺少编号被拒绝'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '预览生成代码缺少编号被拒绝' "$(echo "$out" | head -c 170)"
    fi
    # 预览不存在的表被拒绝
    out=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/preview?id=999999")
    if echo "$out" | grep -q '"code":-1'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '预览不存在的表被拒绝'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '预览不存在的表被拒绝' "$(echo "$out" | head -c 170)"
    fi
    # 下载 zip（校验状态码、内容类型与 zip 魔数 PK）
    gen_zip=$(mktemp)
    zip_meta=$(curl -s -m 20 "${auth_header[@]}" -o "$gen_zip" -w '%{http_code} %{content_type}' "$T/gen/download?id=$gen_id")
    if echo "$zip_meta" | grep -q '^200 application/zip' && [ "$(head -c 2 "$gen_zip")" = "PK" ]; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s %s 字节\n' '下载生成代码 zip' "$(wc -c <"$gen_zip" | tr -d ' ')"
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '下载生成代码 zip' "$zip_meta"
    fi
    rm -f "$gen_zip"
    # 下载缺少编号被拒绝（返回 JSON 错误而不是 zip）
    out=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/download")
    if echo "$out" | grep -q '"code":-1'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '下载生成代码缺少编号被拒绝'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '下载生成代码缺少编号被拒绝' "$(echo "$out" | head -c 170)"
    fi

    # 删除导入记录（同时清理列信息）
    out=$(curl -s -m 10 "${auth_header[@]}" -X DELETE "$T/gen/delete" -H 'Content-Type: application/json' -d "{\"ids\":[$gen_id]}")
    if echo "$out" | grep -q '"code":0'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '删除代码生成配置'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '删除代码生成配置' "$(echo "$out" | head -c 170)"
    fi
    # 删除后列信息一并清理
    col_left=$(curl -s -m 10 "${auth_header[@]}" "$T/gen/column-list?table_id=$gen_id")
    if echo "$col_left" | grep -q '"total":0'; then
      ok=$((ok+1)); printf '\033[32mok\033[0m    %-44s\n' '删除后列信息已清理'
    else
      bad=$((bad+1)); printf '\033[31mFAIL\033[0m  %-44s %s\n' '删除后列信息已清理' "$(echo "$col_left" | head -c 170)"
    fi
  fi
fi

echo
printf '===== 汇总：通过=%s  SQL列名错误=%s  其他失败=%s =====\n' "$ok" "$sqlbad" "$bad"
[ "$sqlbad" -eq 0 ] && [ "$bad" -eq 0 ] || exit 1
