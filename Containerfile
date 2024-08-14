FROM docker.io/library/golang:1.22-alpine AS build
WORKDIR /go/sesame
ADD go.mod go.sum .
RUN go mod verify
ADD . .
RUN go build

FROM scratch
COPY --from=build /go/sesame/sesame /sesame
CMD ["/sesame"]
