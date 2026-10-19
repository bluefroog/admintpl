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

# 全模块 CRUD 断言：create → list(取 id) → detail → update → delete
crud() {
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

req GET '/job/list?pageSize=3'
req GET '/job/list?name=x&groupId=1&status=0'
req GET '/job/detail?id=1'
req GET '/job/check-name?name=x'
req GET '/job-log/list?pageSize=3'
req GET '/job-log/detail?id=1'

req GET '/login-log/list?pageSize=3'
req GET '/login-log/list?loginName=x&status=0'
req GET '/login-log/detail?id=1'
req GET '/operation-log/list?pageSize=3'
req GET '/operation-log/list?title=x&businessType=1&status=0'
req GET '/operation-log/detail?id=1'

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

req GET '/user-online/list?pageSize=3'
req GET '/user-online/detail?id=1'

echo
echo "===== 写接口（全模块 CRUD 往返）====="
crud config     "name=冒烟参数$S"  "{\"name\":\"冒烟参数$S\",\"key\":\"smoke.key.$S\",\"value\":\"v1\",\"type\":\"N\",\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟参数$S改\",\"key\":\"smoke.key.$S\",\"value\":\"v2\",\"type\":\"N\",\"remark\":\"备注2\"}"
crud department "name=冒烟部门$S"  "{\"parent_id\":0,\"name\":\"冒烟部门$S\",\"ancestors\":\"0\",\"sort\":1,\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"parent_id\":0,\"name\":\"冒烟部门$S改\",\"ancestors\":\"0\",\"sort\":2,\"status\":0,\"remark\":\"备注2\"}"
crud dict-type  "name=冒烟字典$S"  "{\"name\":\"冒烟字典$S\",\"type\":\"smoke_type_$S\",\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟字典$S改\",\"type\":\"smoke_type_$S\",\"status\":0,\"remark\":\"备注2\"}"
crud dict-data  "type=smoke_type_$S" "{\"sort\":1,\"label\":\"冒烟标签$S\",\"value\":\"1\",\"type\":\"smoke_type_$S\",\"css_class\":\"\",\"list_class\":\"\",\"is_default\":\"0\",\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"sort\":2,\"label\":\"冒烟标签$S改\",\"value\":\"2\",\"type\":\"smoke_type_$S\",\"css_class\":\"\",\"list_class\":\"\",\"is_default\":\"0\",\"status\":0,\"remark\":\"备注2\"}"
crud job        "name=冒烟任务$S"  "{\"name\":\"冒烟任务$S\",\"group\":\"DEFAULT\",\"executor\":\"default\",\"invoke_target\":\"smoke.run\",\"job_args\":\"\",\"job_kwargs\":\"\",\"cron_expression\":\"0 0 1 * * ?\",\"misfire_policy\":\"1\",\"concurrent\":\"0\",\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"name\":\"冒烟任务$S改\",\"group\":\"DEFAULT\",\"executor\":\"default\",\"invoke_target\":\"smoke.run\",\"job_args\":\"\",\"job_kwargs\":\"\",\"cron_expression\":\"0 0 1 * * ?\",\"misfire_policy\":\"1\",\"concurrent\":\"0\",\"status\":0,\"remark\":\"备注2\"}"
crud menu       "name=冒烟菜单$S"  "{\"name\":\"冒烟菜单$S\",\"parent_id\":0,\"sort\":1,\"path\":\"/smoke$S\",\"component\":\"\",\"query\":\"\",\"route_name\":\"\",\"is_frame\":1,\"is_cache\":0,\"menu_type\":\"M\",\"visible\":\"0\",\"status\":0,\"perms\":\"smoke:list$S\",\"icon\":\"#\"}"  "{\"id\":%ID%,\"name\":\"冒烟菜单$S改\",\"parent_id\":0,\"sort\":2,\"path\":\"/smoke$S\",\"component\":\"\",\"query\":\"\",\"route_name\":\"\",\"is_frame\":1,\"is_cache\":0,\"menu_type\":\"M\",\"visible\":\"0\",\"status\":0,\"perms\":\"smoke:list$S\",\"icon\":\"#\"}"
crud notice     "title=冒烟公告$S" "{\"title\":\"冒烟公告$S\",\"content\":\"公告内容\",\"type\":1,\"status\":0}"  "{\"id\":%ID%,\"title\":\"冒烟公告$S改\",\"content\":\"公告内容2\",\"type\":1,\"status\":0}"
crud post       "name=冒烟岗位$S"  "{\"code\":\"smoke_post_$S\",\"name\":\"冒烟岗位$S\",\"sort\":9,\"status\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"code\":\"smoke_post_$S\",\"name\":\"冒烟岗位$S改\",\"sort\":8,\"status\":0,\"remark\":\"备注2\"}"
crud role       "name=冒烟角色$S"  "{\"key\":\"smoke_role_$S\",\"name\":\"冒烟角色$S\",\"sort\":9,\"status\":0,\"menu_check_strictly\":1,\"dept_check_strictly\":1,\"data_scope\":1,\"del_flag\":0,\"remark\":\"备注1\"}"  "{\"id\":%ID%,\"key\":\"smoke_role_$S\",\"name\":\"冒烟角色$S改\",\"sort\":8,\"status\":0,\"menu_check_strictly\":1,\"dept_check_strictly\":1,\"data_scope\":1,\"del_flag\":0,\"remark\":\"备注2\"}"
crud user       "username=smoke$S" "{\"username\":\"smoke$S\",\"mobile\":\"139$S\",\"nickname\":\"冒烟用户$S\",\"status\":1,\"email\":\"smoke$S@test.com\",\"gender\":1,\"avatar\":\"\",\"departmentID\":0,\"remark\":\"备注1\",\"isAdmin\":false}"  "{\"id\":%ID%,\"username\":\"smoke$S\",\"mobile\":\"139$S\",\"nickname\":\"冒烟用户$S改\",\"status\":1,\"email\":\"smoke$S@test.com\",\"gender\":1,\"avatar\":\"\",\"departmentID\":0,\"remark\":\"备注2\",\"isAdmin\":false}"

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
printf '===== 汇总：通过=%s  SQL列名错误=%s  其他失败=%s =====\n' "$ok" "$sqlbad" "$bad"
[ "$sqlbad" -eq 0 ] && [ "$bad" -eq 0 ] || exit 1
