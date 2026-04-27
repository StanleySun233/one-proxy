cd ./apps/one-panel-api

PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" \
  MYSQL_DSN="${MYSQL_DSN:-root:password@tcp(127.0.0.1:3306)/one_proxy?charset=utf8mb4&parseTime=true&loc=UTC}" \
  air
