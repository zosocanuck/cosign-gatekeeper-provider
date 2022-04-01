IMAGE := "zosocanuck/cosign-gatekeeper-provider:certs"
NAMESPACE := "cosign-gatekeeper-provider"

.PHONY: all docker install unistall

all: docker install

docker:
	docker build -f Dockerfile -t $(IMAGE) .
#	docker push $(IMAGE)

install:
	kubectl create namespace $(NAMESPACE)
	kubectl create secret generic kyverno-chain --from-file=chain=./tests/goodchain.crt -n $(NAMESPACE)
	kubectl apply -f policy/rolebinding-provider.yaml
	kubectl apply -f manifest/ -n $(NAMESPACE)
	kubectl apply -f policy/template.yaml
	sleep 5
	kubectl apply -f policy/constraint.yaml 

uninstall:
	kubectl delete -f manifest/ -n $(NAMESPACE)
	kubectl delete -f policy/template.yaml
	kubectl delete -f policy/constraint.yaml
	kubectl delete -f policy/rolebinding-provider.yaml
	kubectl delete -f secret kyverno-chain -n $(NAMESPACE)
	kubectl delete namespace $(NAMESPACE)

test:
	kubectl apply -f policy/examples/valid.yaml
	kubectl apply -f policy/examples/error.yaml