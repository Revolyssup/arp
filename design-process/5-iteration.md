# Further Optimisations

## Route Matching

Previosly matchers were being created for each Route object and were run by looping over each route. This is bad for a number of reasons. Like I can't use RadixTree or any correlation mechanism and I am having to parse the path n number of times for each route. There are all sort of problems with this approach. Now there are three types of Matchers: PathMatcher, HeaderMatcher, MethodMatcher per HTTPRouter. And as routes are added, these matcher instances update their internal state. So at the time of matching, we just have to call one matcher which can be optimised to return of all matching routes for it.

## Adding LRU Cache in Route matching

Matcher benchmark without cache

```bash
 go test -bench=. -benchmem ./pkg/route/
goos: darwin
goarch: amd64
pkg: github.com/Revolyssup/arp/pkg/route
cpu: VirtualApple @ 2.50GHz
BenchmarkPathMatcher/Size_10-8            309099              3784 ns/op               2 B/op          0 allocs/op
BenchmarkPathMatcher/Size_100-8            29160             41999 ns/op               2 B/op          0 allocs/op
BenchmarkPathMatcher/Size_1000-8            4131            285938 ns/op               2 B/op          0 allocs/op
BenchmarkPathMatcher/Size_5000-8            1654            729198 ns/op              25 B/op          0 allocs/op
```

Matcher benchmark with LRU cache

```bash
 go test -bench=. -benchmem ./pkg/route/
goos: darwin
goarch: amd64
pkg: github.com/Revolyssup/arp/pkg/route
cpu: VirtualApple @ 2.50GHz
BenchmarkPathMatcher/Size_10-8           6232670               199.6 ns/op            80 B/op          3 allocs/op
BenchmarkPathMatcher/Size_100-8          6150762               194.4 ns/op            80 B/op          3 allocs/op
BenchmarkPathMatcher/Size_1000-8         5749389               203.4 ns/op            80 B/op          3 allocs/op
BenchmarkPathMatcher/Size_5000-8         6048776               200.5 ns/op            80 B/op          3 allocs/op
```

## Plugin Interface

The HandleRequest(req) just couldn't work with plugins that need to work with responsewriter before hitting upstream like the response cache plugin. So now this method is changed to HandleRequest(req, res)(final, err) where res is the reponsewriter before it is written to by the upstream so that plugins can act on it. When a plugin returs final as true, no further plugins will be executed.

The WrapResponseWriter callback which only recieves the Response now is changed to HandleResponse(req, res).

The previous abstraction was stupid and I hadn't thought it through. The plugin might need ResponseWriter before request is sent to upstream. And also might need the original request while processing the response from upstream.

## Changes in LRU Cache and Destroy() method in Plugin Interface

LRUCache is instantiated per plugin instance and plugin instances are recreated on config changes. This means that previous plugin instances are no longer used but each plugin instance might be working with some internal or external state which might be referenced by some go routines which might cause both goroutine and memory leak. Destroy() method will be called on the previous pluginchain when new pluginchain is created in httprouter. Plugin chain will call Destroy() on each plugin which will allow it to do cleanup. Example of this cleanup: When responsecache plugin instance is no longer in use, it would want to Reset it's LRUCache to avoid go routine leak from the lrucache. And to propagate this cancellation to LRUCache, a Reset() method is added which will reset the underlying data and cancel all running go routines.
