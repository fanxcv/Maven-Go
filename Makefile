HUB_USER = f763180872
NAME = maven-go

.PHONY: clean

build: clean
	go mod tidy
	go build -o MavenGo src/main.go
	chmod a+x MavenGo
run: build
	./MavenGo
docker: clean
	docker build --no-cache -t $(NAME) .
init: clean
	check=`docker buildx ls | grep ^xBuilder`; \
	if [ "$$check" == "" ];then \
	  docker buildx create --name xBuilder --driver docker-container; \
	  docker buildx use xBuilder; \
	fi
push: init
	cp Dockerfile DockerfileX
	sed -i 's/FROM /FROM --platform=$$TARGETPLATFORM /g' DockerfileX
	docker buildx build --platform linux/arm,linux/arm64,linux/amd64 -t $(HUB_USER)/$(NAME) -f DockerfileX . --push
	rm -rf DockerfileX
clean:
	-docker images | egrep "<none>" | awk '{print $$3}' | xargs docker rmi
	-rm -rf MavenGo go go.sum