package launcher

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func NewECS(c *AWSConfig) *ecs.ECS {
	return ecs.New(NewSession(c))
}

type ECSLauncher struct {
	Client *ecs.ECS
	Wg     sync.WaitGroup
}

func NewECSLauncher(c *AWSConfig) *ECSLauncher {
	return &ECSLauncher{Client: NewECS(c)}
}

func (l *ECSLauncher) Run(tasks []*ECSTask) {
	for _, t := range tasks {
		l.Wg.Add(1)
		go l.run(t)
	}
	l.Wg.Wait()
}

func (l *ECSLauncher) run(task *ECSTask) {
	err := task.Run(l.Client)
	if err != nil {
		l.Wg.Done()
		return
	}

	for {
		select {
		case <-task.doneCh:
			l.Wg.Done()
			return
		default:
			task.Describe(l.Client)
		}
	}
}

type ECSTask struct {
	doneCh chan struct{}
	Input  *ecs.RunTaskInput
	Output *ecs.RunTaskOutput
}

func NewECSTask(input *ecs.RunTaskInput) *ECSTask {
	return &ECSTask{
		doneCh: make(chan struct{}, 1),
		Input:  input,
	}
}

func (t *ECSTask) Run(client *ecs.ECS) error {
	for {
		resp, err := client.RunTask(t.Input)

		if err != nil {
			if strings.Contains(err.Error(), "No Container Instances") {
				fmt.Printf("wait for launching instances:%v\n", t.StringEnvs())
				time.Sleep(30 * time.Second)
				continue
			} else if strings.Contains(err.Error(), "ThrottlingException") {
				fmt.Printf("wait for becoming empty throttles:%v\n", t.StringEnvs())
				time.Sleep(30 * time.Second)
				continue
			} else {
				return err
			}
		}

		if len(resp.Failures) > 0 {
			var isRetry bool

			for _, failure := range resp.Failures {
				reason := aws.StringValue(failure.Reason)
				if strings.HasPrefix(reason, "RESOURCE:") || reason == "AGENT" {
					isRetry = true
				}
			}

			if isRetry {
				fmt.Printf("wait for releasing machine resource:%v\n", t.StringEnvs())
				time.Sleep(30 * time.Second)
				continue
			}
		}

		if len(resp.Tasks) > 0 {
			t.Output = resp
			fmt.Printf("task started:%s\n", t.StringEnvs())
			break
		}

		// errがnilでFailures, Tasksが両方共空の場合もある
		fmt.Printf("fail RunTask():%s\n", t.StringEnvs())
		time.Sleep(30 * time.Second)
	}

	return nil
}

func (t *ECSTask) Describe(client *ecs.ECS) {
	for {
		input := &ecs.DescribeTasksInput{
			Cluster: t.Input.Cluster,
			Tasks:   []*string{t.GetArn()},
		}

		resp, err := client.DescribeTasks(input)
		if err != nil {
			fmt.Printf("%v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if len(resp.Failures) > 0 {
			for _, failure := range resp.Failures {
				fmt.Printf("%v\n", failure)
			}
			time.Sleep(10 * time.Second)
			continue
		}

		if len(resp.Tasks) > 0 {
			task := resp.Tasks[0]
			if aws.StringValue(task.LastStatus) == "STOPPED" {
				var failed bool

				if len(task.Containers) > 0 {
					for _, c := range task.Containers {
						if c.ExitCode != nil && *c.ExitCode > 0 {
							failed = true
							fmt.Printf("fail task exitCode:%v reason:%v containerArn:%v taskArn:%v\n", *c.ExitCode, *c.Reason, *c.ContainerArn, *c.TaskArn)
						}
					}
				}

				if !failed {
					fmt.Printf("finish task arn:%v elappsed:%v envs:%v\n", aws.StringValue(t.GetArn()), task.StartedAt, task.StoppedAt, t.StringEnvs())
				}

				t.doneCh <- struct{}{}

				break
			} else {
				fmt.Printf("task status:%v arn:%v envs:%v\n", aws.StringValue(task.LastStatus), aws.StringValue(t.GetArn()), t.StringEnvs())
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func (t *ECSTask) GetArn() *string {
	return t.Output.Tasks[0].TaskArn
}

func (t *ECSTask) StringEnvs() string {
	var ret []string

	for _, env := range t.Input.Overrides.ContainerOverrides[0].Environment {
		ret = append(ret, fmt.Sprintf("%s:%s", aws.StringValue(env.Name), aws.StringValue(env.Value)))
	}

	return strings.Join(ret, ",")
}
