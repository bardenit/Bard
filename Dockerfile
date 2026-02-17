FROM golang:1.25-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=1 go build -o /app -ldflags="-s -w -X main.BuildTime=${BUILD_TIME}" .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /app /app
COPY templates/ /templates/
COPY static/ /static/
VOLUME /data
ENV DB_PATH=/data/budget.db
ENV TEMPLATE_DIR=/templates
ENV STATIC_DIR=/static
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app"]
