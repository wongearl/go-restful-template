TAG?=dev
AISERVER ?= leovamwong/ai-server:${TAG}
AICONTROLLERMANAGER ?= leovamwong/ai-controller-manager:${TAG}
# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
export GOPROXY=https://goproxy.cn,direct
GOENV  := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
AI_PKG := ai-server
EDITION ?= Commercial
LDFLAGS += -X "$(AAI_PKG)/pkg/version.releaseVersion=$(shell git describe --tags --always)"
LDFLAGS += -X "$(AI_PKG)/pkg/version.buildDate=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(AI_PKG)/pkg/version.gitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(AI_PKG)/pkg/version.gitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
LDFLAGS += -X "$(AI_PKG)/pkg/version.edition=$(EDITION)"
LDFLAGS += -w -s
TOOLEXEC?=

GV="iam.ai.io:v1 core.ai.io:v1"
GO_BUILD := $(GOENV) go build -trimpath -gcflags "all=-N -l" -ldflags '$(LDFLAGS)' $(TOOLEXEC)

##@ Development
jsonschema:
	python3 hack/openapi2jsonschema.py config/crd/bases/*

yaml-check:
	kubeconform -schema-location pkg/api/schema/{{.ResourceKind}}_v1.json -schema-location default -skip v1beta1/DataVolume -ignore-missing-schemas -ignore-filename-pattern config/samples/ai-config.yaml config/samples


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./pkg/api/..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/api/..."

.PHONY: clientset
clientset:
	./hack/generate_client.sh ${GV}

generate-all: generate clientset manifest

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...
test:
	go test ./... -coverprofile coverage.out
	go tool cover -func=coverage.out
tojson:
	sh ./hack/toJson.sh
pre-commit: generate-all fmt vet test

##@ Build

.PHONY: ai-server
ai-server:
	$(GO_BUILD) -o bin/ai-server cmd/ai-server/ai-server.go
.PHONY: run-apiserver
run-aiserver:
	go run cmd/ai-server/ai-server.go
image-build-aiserver:
	docker build . -f build/ai-server/Dockerfile -t ${AISERVER} --build-arg GOPROXY=https://goproxy.cn,direct
image-push-aiserver:
	docker push ${AISERVER}

.PHONY: ai-controller-manager
ai-controller-manager:
	$(GO_BUILD) -o bin/ai-controller-manager cmd/ai-controller-manager/ai-controller-manager.go
.PHONY: run-ai-controller-manager
run-ai-controller-manager:
	go run cmd/ai-controller-manager/ai-controller-manager.go
image-build-ai-controller-manager:
	docker build . -f build/ai-controller-manager/Dockerfile -t ${AICONTROLLERMANAGER} --build-arg GOPROXY=https://goproxy.cn,direct
image-push-ai-controller-manager:
	docker push ${AICONTROLLERMANAGER}
##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${AICONTROLLERMANAGER}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
deploy-with-tag:
	mkdir -p bin && rm -rf bin/config
	cp -r config bin
	cd bin/config/manager && kustomize edit set image leovamwong/ai-server:master=${AICONTROLLERMANAGER} && \
		kustomize edit set image leovamwong/ai-controller-manager:master=${AISERVER}
	export https_proxy=
	kustomize build bin/config/default | kubectl apply -f -
fresh-k3d:
	k3d cluster delete
	k3d cluster create --registry-config config/registry/default.yaml --agents 2 \
		--image 10.121.218.184:30002/cache/rancher/k3s:v1.24.4-k3s1
fresh-k3d-with-tag: fresh-k3d deploy-with-tag

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

print-manifests:
	kustomize build config/default | kubectl apply --dry-run=client -oyaml -f -

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

.PHONY: clean
clean: ## Download kustomize locally if necessary.
	rm -rf bin/ai-server bin/ai-controller-manager

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(shell pwd)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef






