GOCMD=go
GOBUILD=${GOCMD} build
GOCLEAN=${GOCMD} clean
GOTEST=${GOCMD} test
GOGET=${GOCMD} get
VERSION=1.0
DATE= `date +%FT%T%z`


BINARY_NAME="dynamic-sharding"
BINARY_LINUX=${BINARY_NAME}_linux
BUILDUSER="ning1875"
LDFLAGES=" -X 'github.com/prometheus/common/version.BuildUser=${BUILDUSER}'  -X 'github.com/prometheus/common/version.BuildDate=`date`'  "
all:  deps build
deps:
	export GOPROXY=http://goproxy.io
	export GO111MODULE=on
build:
		${GOBUILD}  -v  -ldflags ${LDFLAGES} -o ${BINARY_NAME} pkg/main.go
test:
		${GOTEST} -v ./...
clean:
		${GOCLEAN}
		rm -f ${BINARY_NAME}
		rm -f ${BINARY_LINUX}
run:
		${GOBUILD} -o ${BINARY_NAME} -v ./...
		./${BINARY_NAME}


build-linux:
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${BUILDTOOL}

