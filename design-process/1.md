# First iteration for ARP

Following wil be a rough log of thought process that went into designing ARP. This is not meant to be or structured in the format of a proper design doc.

## Functional Requirement

1. Forward http traffic from client to a dynamically selected upstream server via service discovery.
2. HTTP Traffic will be the main goal to start but we will be extensible enough to handle different L7 protocols later.
3. Traefik has concept of middlewares where each middleware implements ServeHTTP and for responses wraps in its custom ResponseWriter. I dont want that. The API that is exposed for users to add in-tree functionality like traffic split or anything will be via plugins where you only need to implement two methods HandleRequest(req) HandleResponse(res) `Priority() int` where All plugins are chained together and setup internally. Each plugin will be run in order of priority.

## Entities

- **Listener**: Part of static config file and defines two things to begin with: tls and port. When tls is non nil and configured, it will further have path to cert files and will do tls termination. But we will start by handling only pure http traffic and later add support for https.

- **Routes**: Part of dynamic configuration and will have three main fields: listener(name of listener they want to attach to), match(matching criteria), target(name of the upstream). Routes and Listeners have n:n relationship.

- **Upstream**: (1:n with Route) Part of dynamic configuration as well. This will have nodes field which will contain array of endpoint addresses to proxy request to based on another field type which can be "LoadBalancer", "Least weighted", etc. This will also have a discovery field when nodes are not passed explicitly. The single work each discovery implementer needs to do is to populate Route with a set of upstreams. I will think more on this at the end after defining all entities.

- **Plugin**: Go modules loaded in-tree which implements the above said interface. Multiple Plugin instances can be attached to a single route. Relationship is 1:n. And as discussed above, plugins are essentially simplified middlewares.

- **Provider**: This interface is just Provide(chan<- dynamic.Config). This is kind of directly inspired by traefik.

- **Configuration Watcher**: Because I like the throttling mechanism of traefik, this is also kind of inspired from traefik. So data flow will be Provider -> ConfigurationWatcher -> Listener. Configuration watcher will find the diff and figure out routes related to which listener have been modified and will send the data to only that listener using the below mechanism.

- **Event Bus**: I feel like the control logic of a reverse proxy kinda has a vibe of "parts of configuration" flowing from one object to another. I want to have an event bus module that fascilitates all this via simple Publish-Subscribe. The event bus will be generic. Use cases:

1. Configuration Watcher will publish data to topic(listener name) and Each listener instance will Subscribe for configurations meant for it.

2. I am thinking of Service Discovery as nothing but a data source for Upstream.Nodes. Meaning the event bus can be used here as well. So the logic will be like following when a route is added with let's say "Kubernetes" as discovery. Note that the static configuration for discovery implementors will be part of Static config.

a. Inside the logic that processes Routes, it sees discovery type non-empty.
b. For now, each discovery service will have a single instance only. So the logic looks up to see if that instance is running(maybe triggered by other routes), if not it start it with the static discovery config and use the serviceDiscovery event bus. The publisher will be the single running instance and topic will be `discovery type. service name`, in this case lets say "kubernetes.httpbin". Now the kubernetes provider will publish data to this when lets say k8s endpoints change. The subscriber will be the Router which encapsulates the upstream. The data sent over this instance of event bus is "[]Nodes" and whenever a given Router instance(subscriber) detects new []Nodes, it will update it's Upstream. Initially a simple DNS discovery type can be implemented along with the default option to just not have any discovery type and get data from dynamic conf.

- **HTTPRouter**: Each listener will have an instance of HTTPRouter which gets recreated and updated as new dynamic configuration is received by the listener. It combines the Plugins and Upstream to return a single http.Handler interface. This handler has the following data plane flow: Route is evaluated, an internal Router is picked -> All plugins are executed in order -> The ReverseProxy instance gets the upstream.endpoint(picked endpoint), request, response and will be responsible for copying data between upstream and client connection.

*One constraint is: Cannot use httputil.ReverseProxy so we will have our own implementation of ReverseProxy to handle the upstream connection once a node from an upstream is chosen.*

Other things to note:

1. Providers like "File" provider can allow users to remove redundancy by adding a `upstreams:` field and `routes:` referencing the upstream by `upstream_id` or `upstream_name`. But for purposes of simplicity, when the provider creates and sends over dynamic.Configuration, each route will get its own copy of upstream with no reference mapping. This is so that I can have less cognitive load reasoning about data after it enters Listener or HTTPRouter.

2. Initially we will only support File Provider.

3. Kubernetes(in future) can be both a Provider and a Service Discoverer but if it's used as a Provider then it'll be giving all the configuration so separate Service discovery mechanism is not needed for it. Or maybe I decide to keep it simple and not have Kubernetes as a Provider to keep things extremely simple. Fully support K8s Resources or add any CR or support Gateway or ingress API is a non-goal for now.
