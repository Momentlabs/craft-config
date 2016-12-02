repo := craft-controller-test

# AWS configuration
aws_repo := 033441544097.dkr.ecr.us-east-1.amazonaws.com
profile := momentlabs
region := us-east-1

help:
	@echo local \# builds the dockerfile localy to $(rep)
	@echo test-local \# smoke test on the installed binary.
	@echo deploy-to-repo \# builds and pushes the file the AWS repo.

app: 
	cp ../release/craft-config_linux_amd64 .

local: app
	@echo building local image: $(repo) with $(craft-config-latest)
	docker build -t $(repo) .

test: local 
	docker-compose up
	
# Man, this shit has to stop. Either move to rake or look for an alternative.
deploy-to-repo: login := $(shell aws --profile $(profile) --region $(region) --output text ecr get-login )
deploy-to-repo: token := $(shell echo $(login) | awk '{print $$6}')

# @echo Bulding and pushing repository repository:$(repo)
deploy-to-repo: test-local
	@echo logging into AWS before push.
	@docker login -u AWS -p $(token) $(aws_repo)
	docker tag $(repo):latest $(aws_repo)/$(repo):latest
	docker push $(aws_repo)/$(repo):latest
