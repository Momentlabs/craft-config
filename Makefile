repo := craft-config-builder
aws_profile = momentlabs

login := $(shell aws --output text ecr get-login --profile $(aws_profile) --region us-east-1)
token := $(shell echo $(login)| awk '{print $$6}')

local:
	@echo building local image: $(repo)
	docker build -t $(repo) .

deploy-build-container:
	@echo Bulding and pushing repository repository: $(repo)
	@docker login -u AWS -p $(token) https://033441544097.dkr.ecr.us-east-1.amazonaws.com
	docker build -t $(repo) .
	docker tag $(repo):latest 033441544097.dkr.ecr.us-east-1.amazonaws.com/$(repo):latest
	docker push 033441544097.dkr.ecr.us-east-1.amazonaws.com/$(repo):latest

release-build:
	docker-compose up
