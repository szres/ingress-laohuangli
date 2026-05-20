

.PHONY: default down upgrade clean

default:
	docker compose up -d

down:
	docker compose down

upgrade:
	COMPOSE_BAKE=true docker compose build
	docker compose down
	docker compose up -d
	docker image prune -f

clean:
	docker compose down -v --rmi all

backup:
	tar -czf "db-$(date +%F).tar.gz" db
