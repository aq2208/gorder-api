.PHONY: up down migrate run

up:
\tdocker compose -f deployments/docker-compose.dev.yml up -d

down:
\tdocker compose -f deployments/docker-compose.dev.yml down -v

migrate:
\tmysql -h 127.0.0.1 -P 3306 -uroot -proot orders < deployments/migrations.sql

run:
\tHTTP_ADDR=:8080 \
\tMYSQL_DSN="root:root@tcp(127.0.0.1:3306)/orders?parseTime=true" \
\tREDIS_ADDR="127.0.0.1:6379" \
\tgo run ./cmd/order-api
