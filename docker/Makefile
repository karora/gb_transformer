

all: container

.PHONY: clean
clean:
	rm -rf xformer ca-certificates.crt

container: xformer Makefile ca-certificates.crt empty Dockerfile
	docker build -t registry.gitlab.com/karora/gb-xformer:latest .
	docker tag registry.gitlab.com/karora/gb-xformer:latest registry.gitlab.com/karora/gb-xformer:testing

.PHONY: staging
staging:
	docker tag registry.gitlab.com/karora/gb-xformer:testing registry.gitlab.com/karora/gb-xformer:staging
	docker push registry.gitlab.com/karora/gb-xformer:staging

.PHONY: production
production:
	docker tag registry.gitlab.com/karora/gb-xformer:staging registry.gitlab.com/karora/gb-xformer:production
	docker push registry.gitlab.com/karora/gb-xformer:production

xformer: ../xformer
	cp ../xformer .

ca-certificates.crt: /etc/ssl/certs/ca-certificates.crt
	cp /etc/ssl/certs/ca-certificates.crt .

empty:
	mkdir empty
