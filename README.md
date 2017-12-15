# ECS launcher for Golang

Tasks in Amazon ECS launcher library for Golang.

## Install

```
$ go get -u github.com/yoppi/ecs-launcher-go
```

## Usage

```go
package main

import (
  launcher "github.com/yoppi/ecs-launcher-go"
)

func main() {
  l := launcher.NewECSLauncher(&launcher.AWSConfig{})
  l.Run()
}
```
