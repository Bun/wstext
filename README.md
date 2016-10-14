# A WebSocket to text protocol bridge

## TODO

* Simple authentication: only intended for local access for now.

## Dependencies

* https://github.com/gorilla/websocket

## Usage

Usage of wstext:

    -backend string
          Backend address (default "127.0.0.1:6600")
    -bind string
          Bind address (default ":13542")
    -cert string
          TLS certificate
    -key string
          TLS key
    -path string
          WebSocket path (empty: any) (default "/ws")
    -static-dir string
          Serve static files
