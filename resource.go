package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

type resource struct {
	arn          string
	id           string
	accoundID    string
	region       string
	service      string
	resourceType string
	tags         []tag
}

type tag struct {
	Key   string
	Value string
}

type awsController struct {
	config aws.Config
}

func getResourceTypeAndID(str string) (string, string, error) {
	var resourceType, resourceID string
	r := strings.Split(str, ":")
	if len(r) == 1 || (len(r) == 2 && strings.Contains(r[0], "/")) {
		r := strings.Split(str, "/")
		if len(r) == 1 {
			resourceID = r[0]
		} else if len(r) > 1 {
			resourceType = r[0]
			resourceID = strings.Join(r[1:], "/")
		} else {
			return resourceType, resourceID, fmt.Errorf("could not parse resource string: %s", str)
		}
	} else if len(r) == 2 {
		resourceType = r[0]
		resourceID = r[1]
	} else {
		return resourceType, resourceID, fmt.Errorf("could not parse resource string: %s", str)
	}

	return resourceType, resourceID, nil
}

func (a *awsController) getAllResources() ([]resource, error) {
	client := resourcegroupstaggingapi.NewFromConfig(a.config)
	resources := make([]resource, 0, 1000)
	var paginationToken *string
	for {
		out, err := client.GetResources(context.Background(), &resourcegroupstaggingapi.GetResourcesInput{
			ResourcesPerPage: aws.Int32(100),
			PaginationToken:  paginationToken,
		})
		if err != nil {
			return resources, err
		}
		for _, v := range out.ResourceTagMappingList {
			a, err := arn.Parse(*v.ResourceARN)
			if err != nil {
				return resources, err
			}
			tags := make([]tag, 0, len(v.Tags))
			for _, t := range v.Tags {
				tags = append(tags, tag{
					Key:   *t.Key,
					Value: *t.Value,
				})
			}
			rt, id, err := getResourceTypeAndID(a.Resource)
			if err != nil {
				return resources, err
			}
			resource := resource{
				arn:          *v.ResourceARN,
				region:       a.Region,
				accoundID:    a.AccountID,
				service:      a.Service,
				tags:         tags,
				resourceType: rt,
				id:           id,
			}
			resources = append(resources, resource)
		}
		paginationToken = out.PaginationToken
		if *paginationToken == "" {
			break
		}
	}
	return resources, nil
}
