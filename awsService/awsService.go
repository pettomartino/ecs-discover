package awsService

import (
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type Entry struct {
	Name     string
	Host     string
	HostPort int64
}

type awsService struct {
	Cluster     string
	EntryPoints []string
}

type containerInterface interface {
	ListTasks(*ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	DescribeTasks(*ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	ListContainerInstances(*ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error)
	DescribeContainerInstances(*ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
}

type instanceInterface interface {
	DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

func (aS awsService) isEntryPoint(container *string) bool {

	match, err := regexp.MatchString(strings.Join(aS.EntryPoints, "|"), *container)

	if err != nil {
		return false
	}

	return match
}

func (aS awsService) createEntries(service containerInterface, containersIps map[string]string, nextToken *string) ([]Entry, error) {
	tasks, err := service.ListTasks(&ecs.ListTasksInput{Cluster: aws.String(aS.Cluster), NextToken: nextToken})

	if err != nil {
		return nil, err
	}

	describeInput := &ecs.DescribeTasksInput{Cluster: aws.String(aS.Cluster), Tasks: tasks.TaskArns}

	describedTasks, err := service.DescribeTasks(describeInput)

	if err != nil {
		return nil, err
	}

	entries := []Entry{}

	for _, task := range describedTasks.Tasks {
		containerInstance := containersIps[*task.ContainerInstanceArn]
		for _, container := range task.Containers {
			if len(container.NetworkBindings) > 0 && aS.isEntryPoint(container.Name) {
				entries = append(entries, *&Entry{
					Name:     *container.Name,
					Host:     containerInstance,
					HostPort: *container.NetworkBindings[0].HostPort})
			}
		}
	}

	if tasks.NextToken != nil {
		nEntry, err := aS.createEntries(service, containersIps, tasks.NextToken)
		return append(entries, nEntry...), err
	}

	return entries, nil
}

func (aS awsService) createIPContainerMap(ecsSvc containerInterface, ec2Svc instanceInterface) map[string]string {
	maxResults := int64(100)

	input := &ecs.ListContainerInstancesInput{
		Cluster:    aws.String(aS.Cluster),
		MaxResults: &maxResults}

	cInstances, err := ecsSvc.ListContainerInstances(input)

	if err != nil {
		return nil
	}

	//clean up string to get just Id
	ids := []*string{}

	for _, inst := range cInstances.ContainerInstanceArns {
		id := strings.Split(*inst, "/")
		ids = append(ids, aws.String(id[len(id)-1]))
	}

	dcis, err := ecsSvc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(aS.Cluster),
		ContainerInstances: ids})

	if err != nil {
		return nil
	}

	// map InstanceIds
	ec2IDs := []*string{}
	instanceIDContainer := make(map[string]string)

	for _, di := range dcis.ContainerInstances {
		instanceIDContainer[*di.Ec2InstanceId] = *di.ContainerInstanceArn

		ec2IDs = append(ec2IDs, di.Ec2InstanceId)
	}

	instances, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: ec2IDs})

	if err != nil {
		return nil
	}

	instancesIPs := make(map[string]string)

	for i := 0; i < len(instances.Reservations); i++ {

		for k := 0; k < len(instances.Reservations[i].Instances); k++ {
			instance := instances.Reservations[i].Instances[0]
			containerArn := instanceIDContainer[*instance.InstanceId]
			instancesIPs[containerArn] = *instance.PrivateIpAddress
		}

	}

	return instancesIPs
}

func NewAwsService(cluster string, entryPoints []string) *awsService {
	return &awsService{Cluster: cluster, EntryPoints: entryPoints}
}

func (aS awsService) GetEntries(cI containerInterface, iI instanceInterface) ([]Entry, error) {
	containersIps := aS.createIPContainerMap(cI, iI)

	return aS.createEntries(cI, containersIps, nil)
}
