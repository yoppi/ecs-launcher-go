package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	launcher "github.com/yoppi/ecs-launcher-go"
)

func createTasks() []*launcher.ECSTask {
	var tasks []*launcher.ECSTask

	def := "task-name"
	cluster := "cluster-name"
	container := "container-name"

	for i := 0; i < 10; i++ {
		tasks = append(tasks, launcher.NewECSTask(&ecs.RunTaskInput{
			TaskDefinition: aws.String(def),
			Cluster:        aws.String(cluster),
			Count:          aws.Int64(1),
			Overrides: &ecs.TaskOverride{
				ContainerOverrides: []*ecs.ContainerOverride{
					{
						Name: aws.String(container),
						Environment: []*ecs.KeyValuePair{
							{
								Name:  aws.String("index"),
								Value: aws.String(fmt.Sprint(i)),
							},
						},
					},
				},
			},
		}))
	}

	return tasks
}

func main() {
	l := launcher.NewECSLauncher(&launcher.AWSConfig{})
	l.Run(createTasks())
}
