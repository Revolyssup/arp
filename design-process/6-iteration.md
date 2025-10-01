# Road to v0.1.0

## Buffer reuse and ProxyService

io.Copy is replaced with io.CopyBuffer to pass buffer from a sync.Pool. This pool will be initialised on a ProxyService instance which will contain the state to be reused across connections like bufferpool and connection pooling. Connection pooling is needed for implementing custom RoundTripper.

## Support streaming responses

io.Copy* doesn't immediately flush data to underlying responsewriter which is needed in cases of streaming to gradually send out chunks to client instead of all at once. Even io.CopyBuffer with small buffersize doesn't immediately flush to responsewriter so for now a helper function is added that forces the flush after every Write from upstream response.

## Custom RoundTripper

A very naive implementation of round tripper has been added which takes control of underlying socket and writes the generated upstream request directly to it. This will be improved in future. This roundtripper also reuses tcp connections which are also now maintained in a sync.Pool per ReverseProxy instance.

## TODO

- Better error handling and context passing.
- TLS termination support
- More efficient route matching
- Plugins for most common use cases like - traffic split, auth, redirects, circuit breaking etc.
- Support for docker as service discoverer.
