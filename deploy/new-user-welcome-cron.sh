#!/usr/bin/env bash
set -euo pipefail

export TZ=Asia/Shanghai
body=$'感谢访问sub2api中转站，余额已经赠送，欢迎体验！\n\n访问网址：https://sub2api.ai-baby-dance.com/\n微信联系方式：wxl13231529281'

exec /usr/bin/flock -n /var/lock/sub2api-new-user-welcome.lock \
  /usr/bin/docker exec service-sub2api-yiqan5-sub2api-1 \
  /app/sub2api --send-new-user-welcome --new-user-welcome-body "$body" "$@"
