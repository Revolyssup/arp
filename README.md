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
```

## Dynamic Configuration

```yaml
routes:
  - name: route1
    listener: http
    matches:
      - path: "/"
    target: backend1

upstreams:
  - name: backend1
    type: loadbalancer
    nodes:
      - url: http://httpbin.org/headers
      - url: http://httpbin.org/ip
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
