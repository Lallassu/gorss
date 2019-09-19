all: build release clean

build:
	@cd src && go build .

run: build
	@./src/main -theme themes/default.theme -config gorss.conf
	

release:
	@cd src && CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static -s -w" -o ../bin/gorss_linux
	cd src && go build -ldflags "-s -w" -o ../bin/gorss_osx

clean:
	rm -rf main gorss.log gorss.db bin/*
