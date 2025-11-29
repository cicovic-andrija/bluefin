# Bluefin

A web interface for browsing [Subsurface](https://subsurface-divelog.org/) dive
log records. Bluefin parses Subsurface XML databases and provides a hypermedia
interface for them. It also offers a data API.

## Features

- 🗺️ Browse dives organized by trip
- 📊 Detailed dive information display
- 🌍 View dive sites grouped by region
- 📍 Interactive maps for dive locations
- 🏷️ Tag-based organization
- 🏆 Award tracking
- 📱 Responsive design for mobile and desktop clients

## Requirements

- Go 1.22 or later
- Subsurface XML database file

## Build Bluefin

```bash
go build -o bluefin main.go
```

## Run Bluefin

### Build and Run Locally

```bash
DIVELOG_MODE="dev" \
DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" \
go run ./main.go
```

### Run in Production (HTTPS)

```bash
DIVELOG_MODE="prod" \
DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" \
DIVELOG_IP_HOST="0.0.0.0" \
DIVELOG_PORT="443" \
DIVELOG_PRIVATE_KEY_PATH="/path/to/privkey" \
DIVELOG_CERT_PATH="/path/to/pubcert" \
./bluefin
```

### Run in Production (Behind a Reverse Proxy)

```bash
DIVELOG_MODE="prod-proxy-http" \
DIVELOG_DBFILE_PATH="/path/to/subsurfacedata.xml" \
DIVELOG_IP_HOST="127.0.0.1" \
DIVELOG_PORT="52000" \
./bluefin
```

**Note:** Find a `systemd` config example in [`examples/systemd.service`](examples/systemd.service).

### Interrupt / Stop

```bash
pkill -SIGINT bluefin
```

...or `CTRL-C` locally.

## Server Modes

Bluefin supports three server modes:

- **`dev`** - Development mode. Use for local development and testing. Runs on HTTP without TLS, suitable for localhost access only.

- **`prod`** - Production mode with TLS. Use when running `bluefin` as a standalone server with direct HTTPS access. Requires TLS certificate and private key.

- **`prod-proxy-http`** - Production mode behind a reverse proxy. Use when `bluefin` runs behind a reverse proxy (like `nginx`) that handles TLS termination. `bluefin` runs on HTTP and listens on localhost, while the proxy handles HTTPS.

## Configuration

Environment variables:

- `DIVELOG_MODE` - Server mode: `dev`, `prod`, or `prod-proxy-http`
- `DIVELOG_DBFILE_PATH` - Path to Subsurface XML database file
- `DIVELOG_IP_HOST` - IP address to bind to (default: `127.0.0.1`)
- `DIVELOG_PORT` - TCP port to listen on (default: `8080`)
- `DIVELOG_PRIVATE_KEY_PATH` - Path to TLS private key (required for `prod` mode)
- `DIVELOG_CERT_PATH` - Path to TLS certificate (required for `prod` mode)

## Special Tags

Bluefin supports special tags in the format `_key_value` for enhanced metadata processing.

### Dive Site Descriptions

In dive site descriptions, use the `tags:` prefix followed by special tags before the actual description:

```
tags:_region_pacific Beautiful coral reef dive site.
```

**Supported tags:**
- `_region_{value}` - Sets the dive site's region. Supported values include:
  - `europe`, `asia`, `north-america`
  - `atlantic`, `indian`, `pacific`, `mediterranean`, `red-sea`

The region value is mapped to a display name (e.g., `pacific` to "Pacific Ocean"). If no region tag is found, the site defaults to "Unlabeled Region".

### Dive Tags

In dive tags, special tags starting with `_` are processed separately from regular tags:

**Supported tags:**
- `_award_{value}` - Assigns an award to the dive. Supported values include:
  - `1st-dive`, `1st-seawater-dive`, `1st-shark-encounter`, `1st-night-dive`
  - `1st-30m-dive`, `1st-40m-dive`, `1st-wreck-dive`, `1st-wreck-penetration`
  - `cert-owd`, `cert-aowd-nitrox`, `cert-navigation`, `cert-dry`, `cert-deep`, `cert-wreck`
  - `100th-dive`

The award value is mapped to a display name (e.g., `1st-dive` to "First dive!").

Special tags are not displayed as regular tags but are processed to set dive properties like awards.

## Build a Docker Image

Build a Docker image using the provided [`Dockerfile`](Dockerfile):

```bash
docker build -t bluefin:latest .
```

The Dockerfile uses a multi-stage build:
- Builds a statically linked binary in a Go container
- Copies the binary and static assets to a minimal `scratch` image
- Results in a small, secure container image

**Note:** The Dockerfile hardcodes the following environment variables:
- `DIVELOG_MODE=prod-proxy-http`
- `DIVELOG_DBFILE_PATH=/srv/store/subsurfacedata.xml`
- `DIVELOG_IP_HOST=0.0.0.0`

These can be overridden at runtime using `-e` flags if needed.

To run the container, mount your Subsurface XML database file:

```bash
docker run \
  --name divelog-server \ # optional
  --network $NETWORK_NAME \ # optional
  --publish 127.0.0.1:8077:8077 \ # $HOST_IP:$HOST_PORT:$CONTAINER_PORT
  --volume /host/path/to/subsurfacedata.xml:/srv/store/subsurfacedata.xml \
  --env 'DIVELOG_PORT=8077' \
  --restart on-failure:10 \ # optional
  --detach \
  bluefin:latest
```

## Tools

### SDV (Subsurface Decoder Validator)

The `sdv` tool validates and displays the contents of a Subsurface XML database file. It parses the XML and prints all dive data in a structured format, useful for debugging and verifying database integrity.

**Build:**

```bash
cd tools
go build -o sdv sdv.go
```

**Usage:**

```bash
./sdv /path/to/subsurfacedata.xml
```

The tool outputs detailed information about:
- Database header (program and version)
- Dive sites (UUID, name, coordinates, description)
- Geo data (categories and labels)
- Dive trips
- Individual dives (all fields including ratings, tags, equipment, temperatures, etc.)

If the XML file cannot be parsed or contains errors, the tool will exit with an error code.

## License

Open source - see repository for details.
