all: build release clean

VERSION=`git tag |tail -n1`
build:
	@go build -ldflags "-s -w -X $(shell go list)/internal.Version=${VERSION}" ./cmd/gorss/...

run: build
	@./gorss
	

release:
	@mkdir release
	@mkdir dist
	@CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static -s -w -X $(shell go list)/internal.Version=${VERSION}" -o ../release/gorss_linux ./cmd/gorss/...
	@go build -ldflags "-s -w -X $(shell go list)/internal.Version=${VERSION}" -o ../release/gorss_osx ./cmd/gorss/...
	@cp gorss.conf dist/
	@cp themes/default.theme gorss/
	@cp -r themes dist/ 
	@mv release/gorss_linux dist && tar cvfz gorss_linux.tar.gz dist
	@rm dist/gorss_linux
	@mv release/gorss_osx dist && tar cvfz gorss_osx.tar.gz dist
	@rm -rf release dist

clean:
	rm -rf gorss gorss.log gorss.db release dist *.tar.gz
