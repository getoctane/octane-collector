GCLOUD_PROJECT=octane-public
IMAGE_REPO=us.gcr.io/$(GCLOUD_PROJECT)

IMAGE_NAME=octane-collector
IMAGE_VERSION=0.0.1
IMAGE_TAG="$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)"

build:
	go mod vendor
	docker build -t $(IMAGE_TAG) .

push:
	docker push $(IMAGE_TAG)
