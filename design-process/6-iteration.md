# Road to v0.1.0

## Buffer reuse and ProxyService

io.Copy is replaced with io.CopyBuffer to pass buffer from a sync.Pool. This pool will be initialised on a ProxyService instance which will contain the state to be reused across connections like bufferpool and connection pooling. Connection pooling is needed for implementing custom RoundTripper.

## Support streaming responses

io.Copy* doesn't immediately flush data to underlying responsewriter which is needed in cases of streaming to gradually send out chunks to client instead of all at once. Even io.CopyBuffer with small buffersize doesn't immediately flush to responsewriter so for now a helper function is added that forces the flush after every Write from upstream response.

## Custom RoundTripper

A very naive implementation of round tripper has been added which takes control of underlying socket and writes the generated upstream request directly to it. This will be improved in future. This roundtripper also reuses tcp connections which are also now maintained in a sync.Pool per ReverseProxy instance.

## GoWithRecover and upstream-discovery refactor

Inspired by traefik's internally used safe package, a helper function GoWithRecover is put in utils for error handling in cases of panic. Previous the upstream was importing discovery and the relationship was pretty messed up. Now all discovery related code is taken out of upstream package. Now upstream is a dependency of discovery and not the other way around.

## Bug in LRUCache

The cleanup function for a key was cancelled only on the whole Cache reset. But when the same key is Set again or the key is deleted, we still want to stop the cleanup goroutines as either that key no longer exists(deleted) or a new cleanup function exists for it(updated).
I feel uncomfortable having one cleanup goroutine per key. Adding this to the TODO. Maybe a single separate garbage collector go routine that will be listening on Events will be better.

## fix race condition in EventBus

Though I tried to optimise Subcribe() by copying the subscribers slice and releasing the lock. I have realised the following scenario can cause a panic. Subscribe() starts and takes long time due to large number of subscibed channels, Unsubscribe() starts and closes the channel and removes from subscriber list but the Subscribe() is working on copied list so it will still try to send to closed channel.

```go
//older function with potential panic
func (eb *EventBus[T]) Publish(topic string, data T) {
 eb.mx.Lock()
 // Update cache first
 eb.cache[topic] = data

 // Get current subscribers for the topic
 subscribers := make([]chan T, len(eb.subscribers[topic]))
 copy(subscribers, eb.subscribers[topic])
 eb.mx.Unlock()
 for _, ch := range subscribers {
  time.Sleep(2 * time.Second) // simulate large number of subscribers
  select {
  case ch <- data:
   // Successfully sent
  default:
   // Channel is full, log warning but don't block
   eb.log.Infof("WARNING: Channel full for topic %s, dropping message", topic)
  }
 }
}
```

```bash
go test ./pkg/eventbus
panic: send on closed channel

goroutine 7 [running]:
github.com/Revolyssup/arp/pkg/eventbus.(*EventBus[...]).Publish(0x639620, {0x5ec876, 0x5}, {0x5ec66f, 0x4})
        /home/ashish/dev/arp/pkg/eventbus/eventbus.go:76 +0x1d7
github.com/Revolyssup/arp/pkg/eventbus.TestRaceCondition.func1()
        /home/ashish/dev/arp/pkg/eventbus/eventbus_test.go:18 +0x37
created by github.com/Revolyssup/arp/pkg/eventbus.TestRaceCondition in goroutine 6
        /home/ashish/dev/arp/pkg/eventbus/eventbus_test.go:17 +0xff
FAIL github.com/Revolyssup/arp/pkg/eventbus 2.008s
FAIL
```

## Support HTTP2 over cleartext

For http requests, when `http2: true` inside listener then http2 over clear text will be used. `http2: true` will be ignored for https because http2 is supported by default. Note that the roundtripper implemented in ReverseProxy currently only supports http1 and not http2 frames so the upstream request will still be over http1.1

## TODO

- Support HTTP2(ARP<->Upstream)
- More efficient route matching
- Plugins for most common use cases like - traffic split, auth, redirects, circuit breaking etc.
- Support for docker as service discoverer.
- Refactor LRU cache cleanup to be more efficient.
- mTLS support
