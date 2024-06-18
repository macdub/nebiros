SERVER_OUT := NebirosServer
CLI_OUT := Nebiros
WEB_OUT := NebirosWeb
VERSION := "1.0.0"
BUILD_DIRS = Build Build/server Build/cli Build/web

copycfg: build
	rsync -aq --exclude=*.go* Server/Config Build/server/
	rsync -aq Web/Config Build/web/
	rsync -aq Web/static Build/web/
	rsync -aq Web/templates Build/web/

build:
	CGO_ENABLED=0 go build -o Build/server/${SERVER_OUT}-${VERSION} -ldflags="-X main.version=${VERSION}" Server/NebirosServer.go
	CGO_ENABLED=0 go build -o Build/cli/${CLI_OUT}-${VERSION} -ldflags="-X main.version=${VERSION}" Client/CLI/NebirosClientCLI.go
	CGO_ENABLED=0 go build -o Build/web/${WEB_OUT}-${VERSION} -ldflags="-X main.version=${VERSION}" Web/WebServer.go

package: copycfg
	cd Build; tar czf "NebirosServer-${VERSION}.tar.gz" server
	cd Build; tar czf "NebirosCLI-${VERSION}.tar.gz" cli
	cd Build; tar czf "NebirosWeb-${VERSION}.tar.gz" web

install: copycfg
	if [ ! -e "/opt/Nebiros/" ]; then mkdir "/opt/Nebiros"; fi
	cp -r Build/* "/opt/Nebiros/"
	cd "/opt/Nebiros/server"; ln -fs "NebirosServer-${VERSION}" "NebirosServer"
	cd "/opt/Nebiros/cli"; ln -fs "Nebiros-${VERSION}" "Nebiros"
	cd "/opt/Nebiros/web"; ln -fs "NebirosWeb-${VERSION}" "NebirosWeb"

uninstall:
	-@rm -r "/opt/Nebiros"

clean:
	-@rm -r Build

.PHONY: build clean package copycfg install uninstall