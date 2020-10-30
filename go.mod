module github.com/zllovesuki/rmc

go 1.15

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.4.2-0.20200109200802-5947fa1b3e44

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/TheZeroSlave/zapsentry v1.5.0
	github.com/containerd/containerd v1.4.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/getsentry/sentry-go v0.7.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-chi/cors v1.1.1
	github.com/go-playground/validator/v10 v10.4.1
	github.com/go-redis/redis/v7 v7.4.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/johnsto/go-passwordless v0.0.0-20200616130417-d7e95aa614c8
	github.com/joho/godotenv v1.3.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/streadway/amqp v1.0.0
	github.com/stripe/stripe-go/v72 v72.21.0
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/grpc v1.33.1 // indirect
	google.golang.org/protobuf v1.23.0
	gorm.io/driver/postgres v1.0.5
	gorm.io/gorm v1.20.5
	gotest.tools v2.2.0+incompatible // indirect
	moul.io/zapgorm2 v1.0.1
)
