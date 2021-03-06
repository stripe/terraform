package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksPermissionCreate,
		Update: resourceAwsOpsworksPermissionCreate,
		Delete: resourceAwsOpsworksPermissionDelete,
		Read:   resourceAwsOpsworksPermissionRead,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"allow_ssh": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"allow_sudo": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"user_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// one of deny, show, deploy, manage, iam_only
			"level": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)

					expected := [5]string{"deny", "show", "deploy", "manage", "iam_only"}

					found := false
					for _, b := range expected {
						if b == value {
							found = true
						}
					}
					if !found {
						errors = append(errors, fmt.Errorf(
							"%q has to be one of [deny, show, deploy, manage, iam_only]", k))
					}
					return
				},
			},
			"stack_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsOpsworksPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsOpsworksPermissionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DescribePermissionsInput{
		IamUserArn: aws.String(d.Get("user_arn").(string)),
		StackId:    aws.String(d.Get("stack_id").(string)),
	}

	log.Printf("[DEBUG] Reading OpsWorks prermissions for: %s on stack: %s", d.Get("user_arn"), d.Get("stack_id"))

	resp, err := client.DescribePermissions(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				log.Printf("[INFO] Permission not found")
				d.SetId("")
				return nil
			}
		}
		return err
	}

	found := false
	id := ""
	for _, permission := range resp.Permissions {
		id = *permission.IamUserArn + *permission.StackId

		if d.Get("user_arn").(string)+d.Get("stack_id").(string) == id {
			found = true
			d.SetId(id)
			d.Set("id", id)
			d.Set("allow_ssh", permission.AllowSudo)
			d.Set("allow_sodo", permission.AllowSudo)
			d.Set("user_arn", permission.IamUserArn)
			d.Set("stack_id", permission.StackId)
		}

	}

	if false == found {
		d.SetId("")
		log.Printf("[INFO] The correct permission could not be found for: %s on stack: %s", d.Get("user_arn"), d.Get("stack_id"))
	}

	return nil
}

func resourceAwsOpsworksPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.SetPermissionInput{
		AllowSudo:  aws.Bool(d.Get("allow_sudo").(bool)),
		AllowSsh:   aws.Bool(d.Get("allow_ssh").(bool)),
		IamUserArn: aws.String(d.Get("user_arn").(string)),
		StackId:    aws.String(d.Get("stack_id").(string)),
	}

	var resp *opsworks.SetPermissionOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var cerr error
		resp, cerr = client.SetPermission(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				// XXX: handle errors
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
				return resource.RetryableError(cerr)
			}
			return resource.NonRetryableError(cerr)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return resourceAwsOpsworksPermissionRead(d, meta)
}
