package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/Kong/go-pdk"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dgrijalva/jwt-go"
)

var x = make(map[string][]string)

type Config struct {
	S3Endpoint string
	Region     string
}

func New() interface{} {
	return &Config{}
}

func (conf Config) Access(kong *pdk.PDK) {

	x["Content-Type"] = append(x["Content-Type"], "application/json")

	clientID := extractClientIdFromRequest(kong)

	var valid bool

	sess, _ := session.NewSession(&aws.Config{
		Region:           aws.String(conf.Region),
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String(conf.S3Endpoint),
	})

	// Create S3 service client
	client := s3.New(sess)

	result, err := getObject(client, "localstack", aws.String("development/tenant.json"))
	if err != nil {
		kong.Log.Err("error in get object " + string(err.Error()))
		return
	}

	defer result.Body.Close()
	resultBody, err := ioutil.ReadAll(result.Body)
	if err != nil {
		kong.Log.Err(string(err.Error()))
		return
	}

	var tenantList Tenant
	decoder := json.NewDecoder(strings.NewReader(string(resultBody)))
	err = decoder.Decode(&tenantList)
	if err != nil {
		kong.Log.Err("twas an error")
		return
	}
	kong.Log.Err("Tenant name of 1 - " + tenantList[0].CRS_ORGINATOR)

	originator, _ := kong.Request.GetHeader("crs-originator")

	kong.Log.Err("orginator = " + originator)

	for _, element := range tenantList {
		if element.CRS_ORGINATOR == originator {
			for _, id := range element.CLIENT_ID {
				if id == clientID {
					valid = true
					kong.Log.Err("Client found! valid = true")
				}
			}
		}
	}
	if valid {
		kong.Log.Err("Client found! inside valid")
		return
	} else {
		kong.Log.Err("Client Not found! inside else valid")
		kong.Response.Exit(403, "Invalid Client ID: "+clientID+" for crs-orginator : "+originator, x)
		return

	}
}

//decodes the JWT token and extracts the client ID
func extractClientIdFromRequest(kong *pdk.PDK) string {
	tokenString, err := kong.Request.GetHeader("Authorization")

	if err != nil {
		kong.Log.Err(err.Error())
	}
	ss := strings.Split(tokenString, "Bearer")

	if len(ss) == 2 {
		tokenString = strings.TrimSpace(ss[1])
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		kong.Log.Err(string(err.Error()))
	}

	var clientID string
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		clientID = claims["cid"].(string)
		kong.Log.Err("clientID from cid= " + clientID)
	} else {
		kong.Log.Err(string(err.Error()))
	}

	return clientID
}

type Tenant []struct {
	CRS_ORGINATOR      string   `json:"crs-originator"`
	CRSS_WORKFLOW_TYPE string   `json:"crs-workflow-type"`
	TENANT_NAME        string   `json:"tenant-name"`
	CLIENT_ID          []string `json:"client-id"`
}

func getObject(client *s3.S3, bucket string, key *string) (output *s3.GetObjectOutput, err error) {
	request := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    key,
	}

	return client.GetObject(request)
}
