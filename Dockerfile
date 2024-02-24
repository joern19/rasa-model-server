FROM golang:1.22.0 AS build

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /main

FROM scratch

COPY --from=build /main /main

ENTRYPOINT [ "/main" ]
