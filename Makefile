TAG="xlyk/tagliatelle:latest"

.PHONY: build
build:
	docker build -t ${TAG} .

.PHONY: push
push:
	docker push ${TAG}
