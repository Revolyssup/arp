package types

//TODO: Do I need this types.go?

func RouteEventKey(listenerName string) string {
	return "routes_" + listenerName
}

func StreamRouteEventKey(listenerName string) string {
	return "stream_routes_" + listenerName
}

func ServiceDiscoveryEventKey(typ string, serviceName string) string {
	return "sd_" + typ + "_" + serviceName
}
