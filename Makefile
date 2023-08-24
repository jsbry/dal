BINARY=bin/dal
ZIP=bin/dal.zip
ZIPTOOL=build-lambda-zip

PHONY: install
install:
	go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip

PHONY: build
build:
	set GOARCH=amd64&&set GOOS=linux&&go build -o ${BINARY}
	${ZIPTOOL} -o ${ZIP} ${BINARY}

