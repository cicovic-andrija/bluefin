# MyDives

| Soon to be Bluefin.

## Build

```bash
go build -o mydivesrv main.go
```

## Development

```bash
DIVELOG_MODE="dev" DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" go run main.go
```

## Production

```bash
DIVELOG_MODE="prod" \
DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" \
DIVELOG_IP_HOST="0.0.0.0" \
DIVELOG_PORT="443" \
DIVELOG_PRIVATE_KEY_PATH="/path/to/privkey" \
DIVELOG_CERT_PATH="/path/to/pubcert" \
./mydivesrv
```

## Production (Behind a Reverse Proxy)

```bash
DIVELOG_MODE="prod-proxy-http" \
DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" \
DIVELOG_IP_HOST="127.0.0.1" \
DIVELOG_PORT="52000" \
./mydivesrv
```

## Kill

```bash
pkill -SIGINT mydivesrv
```
