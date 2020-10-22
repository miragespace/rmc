proto:
	docker build -t protogen -f Dockerfile.protogen .
	docker run -v `pwd`:/proto protogen --go_out=. spec/*.proto