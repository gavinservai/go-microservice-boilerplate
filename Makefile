# Builds the go service into main, then subsequently builds Docker image
# Doing this allows for a minimally sized Docker image

all: buildservice
	docker build -t nytimes/hello -f Dockerfile .
buildservice:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
