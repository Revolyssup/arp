# Service Discovery related changes

- Single EventBus will be used for all service discoverers with each discoverer publishing nodes on the topic by it's name.

# EventBus changes

The previous implementation of EventBus had a problem. The eventbus is stateless which I thought won't be an issue and for the most part it isn't except when data is Published on a topic before subscribers come up. Currently the eventbus doesn't store data and just sends it to all subscribers. In the aforementioned case, there will be data loss as the publisher will have no subscriber to send data to and will just discard the data. Currently Listeners(subscribers) and ConfigWatcher(Publisher) both are static and we control that subscribers are set up before data is published but we won't be able to guarantee this while using the EventBus for service discovery where Subscribers are dynamically created upstreams which come and go.

## Solution: Caching last used data

I think the EventBus can still be used by storing the last data for each topic. This should be good enough for usage in a reverse proxy where at most places (ConfigWatcher->Listener/ Discoverer -> DynamicUpstream), we only care about the latest configuration. So the behaviour can be modify like following:

1. When a subscriber is added for a topic, by default it gets the data stored in the cache.

2. Cache will contain map[topic]LastData.

3. Publisher will reset the cache before publishing data to existing subscribers.

4. EventBus will also have Unsubscribe method so that dynamic subscribers don't leak channels as they go.

## TODO for Service Discovery

1. Add a real discoverer like docker.

2. Currently the Discoverer pushes all the Nodes(of all services) at once. This can be optimised as follows.

  a. Topic should be <discoverer.service_name> and so that each upstream instance recieves nodes that only concern it. And any filtering is not required on the consumption side.

  b. This way the discoverer will send out a slice of nodes for a given service whenever there's a change.

  c. In future, maybe this can be improved by some mechanism to send diffs because when a service corresponds to large number of endpoints like maybe in case of Kubernetes, one pod change will trigger all of endpoints to be pushed. Though this optimisation is not on the roadmap for v0.1.

## Changes in configwatcher

- Configwatcher will take a custom Processor and will only be responsible for connecting providers and processor via throttling. Now the data flow is Provider -> ConfigWatcher -> Processor -> EventBus -> Listener -> Router.
