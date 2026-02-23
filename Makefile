IMAGE_NAME := "linode/cert-manager-webhook-linode"
IMAGE_TAG := "v0.4.1"

K8S_VERSION := "1.35.0"

OUT := $(shell pwd)/_out

$(shell mkdir -p "$(OUT)")

.DEFAULT_GOAL := build

.PHONY: verify test build clean _out/kubebuilder rendered-manifest.yaml

ENVTEST_K8S_VERSION = ${K8S_VERSION}
ENVTEST = $(shell pwd)/_out/setup-envtest

verify: _out/kubebuilder
	TEST_ASSET_ETCD=_out/kubebuilder/bin/k8s/$(ENVTEST_K8S_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)/etcd \
	TEST_ASSET_KUBECTL=_out/kubebuilder/bin/k8s/$(ENVTEST_K8S_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)/kubectl \
	TEST_ASSET_KUBE_APISERVER=_out/kubebuilder/bin/k8s/$(ENVTEST_K8S_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)/kube-apiserver \
	go test -v

_out/kubebuilder: $(ENVTEST)
	$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir _out/kubebuilder/bin

$(ENVTEST):
	GOBIN=$(shell pwd)/_out go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

test: verify

build:
	docker build --rm -t "${IMAGE_NAME}:${IMAGE_TAG}" -t "${IMAGE_NAME}:latest" .

clean:
	rm -r "${OUT}"

rendered-manifest.yaml:
	helm template \
        --set image.repository=${IMAGE_NAME} \
        --set image.tag=${IMAGE_TAG} \
        deploy/cert-manager-webhook-linode > "${OUT}/rendered-manifest.yaml"
