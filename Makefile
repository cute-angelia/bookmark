.PHONY: up

up:
	git pull origin main
	git add .
	git commit -am "update"
	git push origin master
	@echo "\n 发布中..."

run:
	rm -rf cmd/bookmark/internal/errorcode/code_string.go
	cd cmd/bookmark/internal/errorcode && go generate && cd -
	cd cmd/bookmark/ && go run main.go