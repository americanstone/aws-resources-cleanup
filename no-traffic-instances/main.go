package main

import (
	"context"
	//"flag"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// EC2DescribeInstancesAPI defines the interface for the DescribeInstances function.
// We use this interface to test the function using a mocked service.
type EC2DescribeInstancesAPI interface {
	DescribeInstances(ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// GetInstances retrieves information about your Amazon Elastic Compute Cloud (Amazon EC2) instances.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a DescribeInstancesOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to DescribeInstances.
func GetInstances(c context.Context, api EC2DescribeInstancesAPI, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return api.DescribeInstances(c, input)
}

func DescribeInstancesCmd(region string) []string {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	cfg.Region = region
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	//fmt.Printf("cfg %v", cfg)

	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeInstancesInput{}

	result, err := GetInstances(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
		fmt.Println(err)
		return []string{}
	}
	instances := make([]string, 0)

	for _, r := range result.Reservations {
		for _, i := range r.Instances {
			//fmt.Println("   " + *i.InstanceId)
			instances = append(instances, *i.InstanceId)
		}

	}
	return instances
}

// CWGetMetricDataAPI defines the interface for the GetMetricData function
type CWGetMetricDataAPI interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

// GetMetrics Fetches the cloudwatch metrics for your provided input in the given time-frame
func GetMetrics(c context.Context, api CWGetMetricDataAPI, input *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
	return api.GetMetricData(c, input)
}

func createInput(startTime *time.Time, endTime *time.Time, id string, namespace string,
	metricName string, dimensionName string, dimensionValue string, period int, stat string) *cloudwatch.GetMetricDataInput {
	return &cloudwatch.GetMetricDataInput{
		EndTime:   endTime,
		StartTime: startTime,
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String(id),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String(metricName),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String(dimensionName),
								Value: aws.String(dimensionValue),
							},
						},
					},
					Period: aws.Int32(int32(period)),
					Stat:   aws.String(stat),
				},
			},
		},
	}
}

func DebugInput(goal string, data *cloudwatch.GetMetricDataInput) {
	b, err2 := json.MarshalIndent(data, "", "    ")
	if err2 != nil {
		panic("wring input value!!!")
	}

	fmt.Printf("%s %v\n", goal, string(b))
}

func DebugOutput(goal string, data *cloudwatch.GetMetricDataOutput) {
	b, err2 := json.MarshalIndent(data, "", "    ")
	if err2 != nil {
		panic("wring input value!!!")
	}

	fmt.Printf("%s %v\n", goal, string(b))
}

func main() {
	id := "inst"
	stat := "Sum"
	namespace := "AWS/EC2"
	dimensionName := "InstanceId"
	diffInMinutes := 10800 // minutes 7.5 days
	period := 86400        // 24 hours
	regions := []string{"us-west-2", "us-east-1"}
	for _, region := range regions {
		instances := DescribeInstancesCmd(region)
		fmt.Printf("total EC2 instances %d in region %s \n", len(instances), region)
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic("configuration error, " + err.Error())
		}

		client := cloudwatch.NewFromConfig(cfg)
		startTime := aws.Time(time.Unix(time.Now().Add(time.Duration(-diffInMinutes)*time.Minute).Unix(), 0))
		endTime := aws.Time(time.Unix(time.Now().Unix(), 0))
		fmt.Printf("Request startTime %v endTime %v \n", startTime, endTime)

		//instances = []string{"i-0145c907e4d8ff74c"}

		for _, v := range instances {
			dimensionValue := v
			networkin := createInput(startTime, endTime, id, namespace, "NetworkIn", dimensionName, dimensionValue, period, stat)
			networkout := createInput(startTime, endTime, id, namespace, "NetworkOut", dimensionName, dimensionValue, period, stat)

			//DebugInput("Metrics query", networkin)
			//DebugInput("Metrics query", networkout)

			resultin, err1 := GetMetrics(context.TODO(), client, networkin)
			resultout, err2 := GetMetrics(context.TODO(), client, networkout)
			if err1 != nil || err2 != nil {
				fmt.Println("Could not fetch metric data")
			} else {
				//DebugOutput("Query Results", resultin)
				//DebugOutput("Query Results", resultout)
				sumin := 0.0
				for _, data := range resultin.MetricDataResults[0].Values {
					sumin += data
				}

				sumout := 0.0
				for _, data := range resultout.MetricDataResults[0].Values {
					sumout += data
				}
				fmt.Printf("instantId: %s total NetworkIn %f GiB NetworkOut %f GiB\n", dimensionValue, float64(sumin/1024/1024/1024), float64(sumout/1024/1024/1024))
			}
		}
	}

}

//metricName := flag.String("mN", "", "The name of the metric") // "NetworkIn|NetworkOut"
// metricName := "NetworkIn"
//namespace := flag.String("n", "", "The namespace for the metric") // "AWS/EC2"
// namespace := "AWS/EC2"
// dimensionName := flag.String("dn", "", "The name of the dimension") // "InstanceId"
// dimensionName := "InstanceId"
//dimensionValue := flag.String("dv", "", "The value of the dimension") // instance_id
// dimensionValue :=
//id := flag.String("id", "", "A short name used to tie this object to the results in the response")// inst
// id := "inst"
//diffInMinutes := flag.Int("dM", 0, "The difference in minutes for which the metrics are required")// 2000
//diffInMinutes := 2000
//stat := flag.String("s", "", "The statistic to to return") // "sum" 2022-06-01T00:00:00.000Z 2022-09-01T00:00:00.000Z
//stat := "sum"
//period := flag.Int("p", 0, "The granularity, in seconds, of the returned data points") //86400
//period := 86400
//flag.Parse()

// if *metricName == "" || *namespace == "" || *dimensionName == "" || *dimensionValue == "" || *id == "" || *diffInMinutes == 0 || *stat == "" || *period == 0 {
// 	fmt.Println("You must supply a metricName, namespace, dimensionName, dimensionValue, id, diffInMinutes, stat, period")
// 	return
// }
