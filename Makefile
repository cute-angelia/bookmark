.PHONY: up

up:
	git pull origin main
	git add .
	git commit -am "update"
	git push origin main
	@echo "\n 发布中..."

run:
	rm -rf cmd/bookmark/internal/errorcode/code_string.go
	cd cmd/bookmark/internal/errorcode && go generate && cd -
	cd cmd/bookmark/ && go run main.go

tag:
	git pull origin main
	git add .
	git commit -am "update"
	git push origin main
	git tag v1.0.9
	git push --tags
	@echo "\n tags 发布中..."