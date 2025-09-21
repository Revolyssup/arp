#!/bin/bash

# Base directory
BASE_DIR="arp"

# Create directories
mkdir -p $BASE_DIR/cmd/arp
mkdir -p $BASE_DIR/internal/config
mkdir -p $BASE_DIR/internal/listener
mkdir -p $BASE_DIR/internal/router
mkdir -p $BASE_DIR/internal/plugin
mkdir -p $BASE_DIR/internal/upstream
mkdir -p $BASE_DIR/internal/proxy
mkdir -p $BASE_DIR/internal/eventbus
mkdir -p $BASE_DIR/internal/watcher
mkdir -p $BASE_DIR/internal/provider/file
mkdir -p $BASE_DIR/internal/provider/kubernetes

# Create files
touch $BASE_DIR/cmd/arp/main.go

touch $BASE_DIR/internal/config/static.go
touch $BASE_DIR/internal/config/dynamic.go

touch $BASE_DIR/internal/listener/listener.go
touch $BASE_DIR/internal/listener/manager.go

touch $BASE_DIR/internal/router/router.go
touch $BASE_DIR/internal/router/httprouter.go

touch $BASE_DIR/internal/plugin/plugin.go
touch $BASE_DIR/internal/plugin/registry.go

touch $BASE_DIR/internal/upstream/upstream.go
touch $BASE_DIR/internal/upstream/loadbalancer.go

touch $BASE_DIR/internal/proxy/reverseproxy.go

touch $BASE_DIR/internal/eventbus/eventbus.go

touch $BASE_DIR/internal/watcher/watcher.go

touch $BASE_DIR/internal/provider/provider.go
touch $BASE_DIR/internal/provider/file/file.go
touch $BASE_DIR/internal/provider/kubernetes/kubernetes.go
