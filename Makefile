test: e2e-test unit-test

unit-test:
	go test ./pkg/... ./cmd/manager/... ./cmd/kuberik/...

e2e-test:
	./scripts/test-e2e.sh
