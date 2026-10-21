.PHONY: go-run go-test go-lint web-dev web-check

go-run:
	$(MAKE) -C citypark-go run

go-test:
	$(MAKE) -C citypark-go test

go-lint:
	$(MAKE) -C citypark-go lint

web-dev:
	cd citypark-modern && npm run dev

web-check:
	cd citypark-modern && npm run check
