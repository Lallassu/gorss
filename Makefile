all: build

VERSION=`git tag |tail -n1`
build:
	@go build -ldflags "-s -w -X $(shell go list)/internal.Version=${VERSION}" ./cmd/gorss/...

run: build
	@./gorss
	

release:
	@mkdir release
	@mkdir dist
	@GOARCH=amd64 GOOS=linux go build -ldflags "-s -w -X $(shell go list)/internal.Version=${VERSION}" -o ./release/gorss_linux ./cmd/gorss/...
	@GOARCH=amd64 GOOS=darwin go build -ldflags "-s -w -X $(shell go list)/internal.Version=${VERSION}" -o ./release/gorss_osx ./cmd/gorss/...
	@cp gorss.conf dist/
	@cp themes/default.theme dist/
	@cp -r themes dist/ 
	@mv release/gorss_linux dist && tar cvfz gorss_linux.tar.gz dist
	@rm dist/gorss_linux
	@mv release/gorss_osx dist && tar cvfz gorss_osx.tar.gz dist
	@rm -rf release dist

clean:
	rm -rf gorss gorss.log gorss.db release dist *.tar.gz
