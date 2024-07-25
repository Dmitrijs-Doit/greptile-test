PORT ?= 8082
PROJECT ?= doitintl-cmp-dev

run:
	GOOGLE_CLOUD_PROJECT=${PROJECT} PORT=${PORT} go run ./cmd/main.go

dev:
	GOOGLE_CLOUD_PROJECT=${PROJECT} PORT=${PORT} yarn g:runAs "air"

dev-prod:
	GOOGLE_CLOUD_PROJECT="me-doit-intl-com" PORT=${PORT} yarn g:runAs "arelo -p '**/*.go' -i '**/.*' -i '**/*_test.go' -- go run ./cmd/main.go"

install:
	go install github.com/makiuchi-d/arelo@latest
	go install github.com/cosmtrek/air@latest

test:
	go install gotest.tools/gotestsum@latest
	yarn test


test-ci:
	go install gotest.tools/gotestsum@latest
	yarn ci


trigger-build:
	../../../scripts/cloudbuild/trigger-cloudbuild.sh "$PWD" "appengine-scheduled-tasks" "us-central1"
