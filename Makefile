proto:
	docker build -t protogen -f Dockerfile.protogen .
	rm -rf spec/*.pb.go
	docker run -v `pwd`:/proto protogen --go_out=. spec/*.proto