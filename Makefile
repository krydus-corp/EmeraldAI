#! Environment file to use when deploying - see .env.example
#! DO NOT change this variable; instead, to change the environment, run the Makefile as `ENV=config/.env Make <command>`
ENV 	?= config/.env.example
VERSION := $(shell git describe --tags)
COMMIT 	:= $(shell git rev-parse HEAD)
SWAGGER_SPEC := ./pkg/services/portal/swaggerui/embed/swagger.json

AWS_ECR := <TBD>
AWS_ROUTE53_ROOT_DOMAIN=emeraldai-dev.com
AWS_ROUTE53_ROOT_DOMAIN_HOSTED_ZONE_ID=Z03577132G6FUSS5R4UJX
AWS_CF_STACK_POLICY_FILE := https://emld-configuration-store.s3.amazonaws.com/aws_cf_stack_policy.json

CONFIGURATION_STORE=emld-configuration-store

# If deploying locally e.g. MONGO_URI=mongodb://mongo:27017, set to `localdb`.
# If deploying using Atas e.g. MONGO_URI=mongodb+srv://root:emld_mongo01@stage-serverless.cqcqn.mongodb.net/?retryWrites=true&w=majority, keep empty.
MONGO_PROFILE ?= ""

include $(ENV)
export
export AWS_PAGER="" # disable aws cli default routing to less

# Enforce pre-commit usage
_ = $(shell pre-commit install)

.PHONY: all
all: help

.PHONY: VERSION
version:
	@echo $(VERSION)

########## High Level Directives ##########

.PHONY: build # Build app services (docker)
build: build-docker

.PHONY: build-arm # Build app services arm64 (docker)
build-arm: build-docker-arm

.PHONY: deploy ## Deploy to ECS
deploy: build push ecs-generate ecs-deploy

.PHONY: deploy-arm ## Deploy to ECS cross compile
deploy-arm: build-docker-arm push ecs-generate ecs-deploy

.PHONY: update ## Update ECS cluster
update: build push ecs-generate ecs-update

.PHONY: generate ## Generate cloudformation template
generate: build push ecs-generate

########## ##################### ##########

.PHONY: update-pkgs
update-pkgs:
	@printf "\033[36m==> %s\033[0m\n" "Updating dependency versions..."
	@go get -u ./...
	@go mod tidy

.PHONY: build-local
build-local: swag lint ## Build app services (local)
	@printf "\033[36m==> %s\033[0m\n" "Building services (local)..."
	@GOBIN="${CURDIR}/build/bin/$(shell uname)" go install -ldflags="-X 'gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal.BuildVersion=$(VERSION)' -X 'gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal.GitCommit=$(COMMIT)'" ./...

.PHONY: build-docker
build-docker: swag lint ## Build app services [amd64] (docker)
	@printf "\033[36m==> %s\033[0m\n" "Building services (Docker)..."
	@docker --context desktop-linux build --build-arg buildno=$(VERSION) --build-arg gitcommithash=$(COMMIT) -f docker/Dockerfile.base --rm -t emerald/builder .
	@docker-compose -f docker/docker-compose.local.yml build --force-rm --no-cache

.PHONY: build-docker-arm
build-docker-arm: swag lint ## Build app services (docker)
	@printf "\033[36m==> %s\033[0m\n" "Building services (Docker)..."
	@docker --context desktop-linux build --build-arg buildno=$(VERSION) --build-arg gitcommithash=$(COMMIT) -f docker/Dockerfile.base-arm64 --rm -t emerald/builder .
	@docker-compose -f docker/docker-compose.local.yml build --force-rm --no-cache

.PHONY: push
push: ## Push services to ECR
	@printf "\033[36m==> %s\033[0m\n" "Pushing services to container registry..."
	@aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR)
	@docker-compose -f docker/docker-compose.local.yml push

.PHONY: submodule
submodule: ## Update submodules
	@git submodule update --init --recursive --remote

.PHONY: up
up: config soft-check ## Start app services (local)
	@printf "\033[36m==> %s\033[0m\n" "Spinning up services (local)..."
	@docker-compose --profile $(MONGO_PROFILE) -f docker/docker-compose.local.yml up -d

.PHONY: down
down: ## Stop and remove app services (local)
	@printf "\033[36m==> %s\033[0m\n" "Spinning down services (local)..."
	@docker-compose -f docker/docker-compose.local.yml rm --stop

.PHONY: test
test: ## Run unit tests (local)
	@go test -short -v ./... -cover -count=1

.PHONY: clean
clean: ## Clean generated build files (local)
	@rm -rf build

.PHONY: swag
swag: ## Build swagger documentation (local)
	@printf "\033[36m==> %s\033[0m\n" "Generating Swagger docs..."
	@swagger generate spec --scan-models -o ./pkg/services/portal/swaggerui/embed/swagger.json

.PHONY: migrate
migrate: ## Migrate the DB (local)
	@docker --context desktop-linux exec -it portal ./migrate

.PHONY: ecs-generate
ecs-generate: config ## Generate ECS service spec (remote)
	@printf "\033[36m==> %s\033[0m\n" "Generating service spec (ECS)..."
	@aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR)
	@docker context create ecs emld-ecs-context --from-env 2>/dev/null; true
	@docker --context emld-ecs-context compose -f docker/docker-compose.ecs.yml --project-name emerald-$(CLUSTER_ENV) --profile $(MONGO_PROFILE) convert > /tmp/emld_cf.yml

.PHONY: ecs-deploy
ecs-deploy: check ## Deploy ECS service specification (Cloudformation Stack) (remote)
	@printf "\033[36m==> %s\033[0m\n" "Deploying service spec (ECS)..."
	@aws cloudformation create-stack --template-body file:///tmp/emld_cf.yml --stack-name emerald-$(CLUSTER_ENV) --tags Key=environment,Value=emerald-$(CLUSTER_ENV) Key=owner,Value=$(OWNER) --capabilities CAPABILITY_IAM --stack-policy-url $(AWS_CF_STACK_POLICY_FILE)
	@aws cloudformation wait stack-create-complete --stack-name emerald-$(CLUSTER_ENV)
	@./scripts/attach_hosted_zone.sh $(CLUSTER_ENV) $(AWS_ROUTE53_ROOT_DOMAIN) $(AWS_ROUTE53_ROOT_DOMAIN_HOSTED_ZONE_ID) CREATE
	@printf "\033[36m==> %s\033[0m\n" "Deployment complete! Visit API docs at $(CLUSTER_ENV).$(AWS_ROUTE53_ROOT_DOMAIN)"

.PHONY: attach
attach:
	@./scripts/attach_hosted_zone.sh $(CLUSTER_ENV) $(AWS_ROUTE53_ROOT_DOMAIN) $(AWS_ROUTE53_ROOT_DOMAIN_HOSTED_ZONE_ID) CREATE
	@printf "\033[36m==> %s\033[0m\n" "Deployment complete! Visit API docs at $(CLUSTER_ENV).$(AWS_ROUTE53_ROOT_DOMAIN)"

.PHONY: ecs-update
ecs-update: ## Update Emerald service specification (Cloudformation Stack) (remote)
	@printf "\033[36m==> %s\033[0m\n" "Updating services (ECS)..."
	@aws cloudformation update-stack --template-body file:///tmp/emld_cf.yml --stack-name emerald-$(CLUSTER_ENV) --tags Key=environment,Value=emerald-$(CLUSTER_ENV) Key=owner,Value=$(OWNER) --capabilities CAPABILITY_IAM --stack-policy-url $(AWS_CF_STACK_POLICY_FILE)
	@aws cloudformation wait stack-update-complete --stack-name emerald-$(CLUSTER_ENV)

.PHONY: ecs-down
ecs-down: check ## Teardown Emerald service specification (Cloudformation Stack) (remote)
	@printf "\033[36m==> %s\033[0m\n" "Stopping services (ECS)..."
	@./scripts/attach_hosted_zone.sh $(CLUSTER_ENV) $(AWS_ROUTE53_ROOT_DOMAIN) $(AWS_ROUTE53_ROOT_DOMAIN_HOSTED_ZONE_ID) DELETE
	@aws cloudformation delete-stack --stack-name emerald-$(CLUSTER_ENV)

.PHONY: ecs-describe
ecs-describe: ## Get Emerald service specification information (ECS)
	@aws cloudformation describe-stacks --stack-name emerald-$(CLUSTER_ENV) | jq ".Stacks [0]"

.PHONY: empty-s3-bucket
empty-s3-bucket: ## Empty and remove user bucket
	@aws s3 rm s3://$(USER_DATA_S3_BUCKET) --recursive
	@aws s3 rb s3://$(USER_DATA_S3_BUCKET) --force

.PHONY: config
config: envsubst ## Apply configuration based on environment (dev, test, stage, prod) and push configs to configuration store
	@printf "\033[36m==> %s\033[0m\n" "Generating and uploading service configs"
	@aws s3 cp ./build/config* s3://emld-configuration-store --recursive --exclude "*" --include "*.yml"
	@aws s3 cp config/casbin* s3://emld-configuration-store --recursive
	@aws s3 cp config/deploy* s3://emld-configuration-store --recursive

.PHONY: envsubst
envsubst:
	@mkdir -p ./build/config
	@printf "\033[36m==> %s\033[0m\n" "Applying cluster environment: $(CLUSTER_ENV)"
	@envsubst < config/config.portal.template.yml > "./build/config/config.portal.$(CLUSTER_ENV).yml"
	@envsubst < config/config.model.template.yml > "./build/config/config.model.$(CLUSTER_ENV).yml"
	@envsubst < config/config.exporter.template.yml > "./build/config/config.exporter.$(CLUSTER_ENV).yml"

.PHONY: check
check:
	@printf "\033[36m==> %s\033[0m\n" "Warning: you are about to modify a production AWS ECS Cluster. If deploying, costs will be incurred (cluster=$(CLUSTER_ENV)); continue? [Y/n]"
	@read line; if [ $$line = "n" ]; then echo aborting; exit 1 ; fi
	@printf "\033[36m==> %s\033[0m\n" "Are you sure? [Y/n]"
	@read line; if [ $$line = "n" ]; then echo aborting; exit 1 ; fi
	@printf "\033[36m==> %s\033[0m\n" "100% positive? [Y/n]"
	@read line; if [ $$line = "n" ]; then echo aborting; exit 1 ; fi
	@printf "\033[36m==> %s\033[0m\n" "OK, seems you know what you are doing..."

.PHONY: soft-check
soft-check:
	@printf "\033[36m==> %s\033[0m\n" "Deploying locally to cluster=$(CLUSTER_ENV); continue? [Y/n]"
	@read line; if [ $$line = "n" ]; then echo aborting; exit 1 ; fi

.PHONY: chglog
chglog: ## Generate CHANGELOG
	@git-chglog  --no-case v0.1.0.. > CHANGELOG.md

.PHONY: loc
loc: ## Generate LOC
	@echo "# LOC\n" > /tmp/LOC.md
	@find . -name '*.go' | xargs wc -l >> /tmp/LOC.md
	@sed "s/^[ \t]*//" /tmp/LOC.md > LOC.md

.PHONY: lint
lint: ## Lint the project
	go fmt ./pkg/...
	go vet ./pkg/...
	staticcheck ./pkg/...
	swagger validate $(SWAGGER_SPEC) --quiet

.PHONY: genswag
genswag: ## Generate Swagger SDKs
	@[ "${lang}" ] || ( echo ">> lang is not set; e.g. 'lang=go'"; exit 1 )
	@mkdir -p sdk/$(lang)
	swagger-codegen generate -i ./pkg/services/portal/swaggerui/embed/swagger.json -l $(lang) -o ./sdk/$(lang)/

.PHONY: lambda-build
lambda-build: ## Build lambda functions
	@envsubst < config/lambda.cf.template.yml > "./build/config/lambda.$(CLUSTER_ENV).yml"
	sam build --region $(AWS_REGION) --template-file './build/config/lambda.$(CLUSTER_ENV).yml' --build-dir ./build/.aws-sam --no-cached
	@chmod -R 777 ./build
	sam package --region $(AWS_REGION) --template-file ./build/.aws-sam/template.yaml --s3-bucket $(CONFIGURATION_STORE) --s3-prefix lambda-deployments --output-template-file ./build/.aws-sam/packaged.yml --force-upload

.PHONY: lambda-deloy
lambda-deploy: config lambda-build ## Deploy lambda functions
	sam deploy --region $(AWS_REGION) --stack-name emld-lambda-$(CLUSTER_ENV) --template-file ./build/.aws-sam/packaged.yml --s3-bucket $(CONFIGURATION_STORE) --s3-prefix lambda-deployments --capabilities CAPABILITY_IAM  --on-failure DELETE --force-upload

.PHONY: lambda-delete
lambda-delete: ## Delete lambda functions
	sam delete --region $(AWS_REGION) --stack-name emld-lambda-$(CLUSTER_ENV) --s3-bucket $(CONFIGURATION_STORE) --s3-prefix lambda-deployments

.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
