package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lorenzodisidoro/bitcoin-butler/s3wrapper"
	"github.com/lorenzodisidoro/bitcoin-butler/walletlive"
)

// Region AWS
const Region = "eu-west-1"

type lambdaResponse struct {
	Status  int    `json:"status"`
	Address string `json:"address"`
	Network string `json:"network"`
}

type lambdaError struct {
	Status  int   `json:"status"`
	Message error `json:"message"`
}

// main call lambda.Start() and pass lambda handler function
func main() {
	lambda.Start(lambdaHandler)
}

// lambdaHandler retrieve environment variables and create a new bitcoin address
// function build and return APIGatewayProxyResponse object to be returned by API Gateway for the request
func lambdaHandler(req events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {
	requestIP := req.RequestContext.Identity.SourceIP
	fmt.Println("Request IP: ", requestIP)

	if req.HTTPMethod == "GET" {
		network, isPresent := os.LookupEnv("NETWORK")
		if !isPresent {
			serverError(http.StatusNotFound, errors.New("Not found environment variable 'NETWORK'"))
		}

		xPub, isPresent := os.LookupEnv("XPUB")
		if !isPresent {
			serverError(http.StatusNotFound, errors.New("Not found environment variable 'XPUB'"))
		}

		path, isPresent := os.LookupEnv("PATH")
		if !isPresent {
			serverError(http.StatusNotFound, errors.New("Not found environment variable 'PATH'"))
		}

		// TODO: Move on S3 an incremental index
		bucket, isPresent := os.LookupEnv("BUCKET_NAME")
		if !isPresent {
			serverError(http.StatusNotFound, errors.New("Not found environment variable 'BUCKET_NAME'"))
		}

		indexFile, isPresent := os.LookupEnv("INDEX_FILE_NAME")
		if !isPresent {
			serverError(http.StatusNotFound, errors.New("Not found environment variable 'BUCKET_NAME'"))
		}

		s3 := &s3wrapper.S3Wrapper{}
		err := s3.New(Region)
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		// get index
		index, err := getAddressIndex(s3, bucket, indexFile)
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		// generate address
		wallet := &walletlive.WalletLive{}
		wallet.New(xPub, path, network)
		address, err := wallet.DeriveAddress(uint32(index))
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		// make response
		response := &lambdaResponse{
			Status:  http.StatusOK,
			Address: address,
			Network: network,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		strResp := string(jsonResponse)

		// update index
		index++
		err = s3.UploadObject(bucket, indexFile, strconv.Itoa(index))
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       strResp,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnauthorized,
	}, nil
}

func getAddressIndex(s3 *s3wrapper.S3Wrapper, bucket, indexFile string) (int, error) {
	objectMap, err := s3.GetObject(bucket, indexFile)
	if err != nil {
		// init bucket and/or index file if not exists
		if err == s3wrapper.ErrNoSuchBucket {
			err = s3.CreateBucket(bucket)
			if err != nil {
				return -1, err
			}
		}

		if err == s3wrapper.ErrNoSuchKey {
			err = s3.UploadObject(bucket, indexFile, string(0))
			if err != nil {
				return -1, err
			}

			return 0, nil
		}

		return -1, err
	}

	index := objectMap[0][0]
	indexInt := 0
	if strings.Index(index, " ") == -1 {
		indexInt, err = strconv.Atoi(index)
		if err != nil {
			return -1, err
		}
	}

	return indexInt, nil
}

// serverError create new response to be returned by API Gateway for the request
func serverError(status int, err error) (events.APIGatewayProxyResponse, error) {
	basicError := events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       http.StatusText(http.StatusInternalServerError),
	}

	responseError := &lambdaError{}
	jsonResponseError, err := json.Marshal(responseError)
	if err != nil {
		return basicError, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(jsonResponseError),
	}, nil
}
