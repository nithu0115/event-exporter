all: container

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
BINARY_NAME = event-exporter

PREFIX = nithmu
IMAGE_NAME = k8s-event-exporter
TAG = v0.1.0

build:
	${ENVVAR} go build -a -o ${BINARY_NAME}

test:
	${ENVVAR} go test ./...

container: build
	docker build --pull -t ${PREFIX}/${IMAGE_NAME}:${TAG} .

push: container
	gcloud docker -- push ${PREFIX}/${IMAGE_NAME}:${TAG}

clean:
	rm -rf ${BINARY_NAME}