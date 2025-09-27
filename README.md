# ARP - Another Reverse Proxy

## Static configuration

```yaml
listeners:
  - name: http
    port: 8080

providers:
  - name: file
    type: file
    config:
      path: "./dynamic.yaml"
discovery:
  - type: demo
    config:
      interval: 10s
```

## Dynamic Configuration

```yaml
routes:
  - name: route1
    listener: http
    matches:
      - path: /ip
    upstream:
      discovery:
        type: demo
    plugins:
      - name: demo2
  - name: route2
    listener: http
    matches:
      - path: /headers
    upstream:
      name: backend1
    plugins:
      - name: demo
upstreams:
  - name: backend1
    nodes:
      - url: https://httpbin.org/headers
      # - url: http://mockbin.org/headers
plugins:
  - name: demo
    type: demo
    config:
      message: "Hello from demo plugin!"
  - name: demo2
    type: demo
    config:
      message: "Hello from demo2 plugin!"
      
```

### Usage

```bash
ARP_CONFIG=./static.yaml ./arp
```

```bash
curl localhost:8080/
{
  "headers": {
    "Accept": "*/*",
    "Accept-Encoding": "gzip",
    "Host": "httpbin.org",
    "User-Agent": "curl/8.13.0",
    "X-Amzn-Trace-Id": "Root=1-68d02bee-7ada1053448207955dc981b6"
  }
}

```
