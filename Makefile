
.PHONY: build
build:
	sh scripts/build.sh

.PHONY: build-scm
build-scm:
	sh scripts/build.sh
	docker build -t wchstu/mydocker:test .
	docker push wchstu/mydocker:test

.PHONY: run
run: 
	kubectl delete po mydocker -n dev
	kubectl apply -f ./scripts/deploy.yaml