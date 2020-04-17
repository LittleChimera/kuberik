test: e2e-test unit-test

unit-test:
	go test \
	./pkg/... \
	./cmd/manager/... \
	./cmd/kuberik/...

e2e-test:
	operator-sdk test local ./test/e2e \
    --no-setup \
    --namespace default \
    --up-local
