IMAGE := "zosocanuck/cosign-gatekeeper-provider:certs"

.PHONY: all docker install unistall

all: docker install

docker:
	docker build -f Dockerfile -t $(IMAGE) .
#	docker push $(IMAGE)

install:
	kubectl apply -f manifest/
	kubectl apply -f policy/template.yaml
	sleep 5
	kubectl apply -f policy/constraint.yaml 

uninstall:
	kubectl delete -f manifest/
	kubectl delete -f policy/template.yaml
	kubectl delete -f policy/constraint.yaml
	kubectl delete -f policy/examples/valid.yaml
	kubectl delete -f policy/examples/error.yaml

test:
	kubectl apply -f policy/examples/valid.yaml
	kubectl apply -f policy/examples/error.yaml