package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
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
	Status  int    `json:"status"`
	Message string `json:"message"`
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
		// retrieves the value of the environment variables
		network, isPresent := os.LookupEnv("NETWORK")
		if !isPresent {
			return serverError(http.StatusNotFound, errors.New("Not found environment variable 'NETWORK'"))
		}

		encryptedXPub, isPresent := os.LookupEnv("XPUB")
		if !isPresent {
			return serverError(http.StatusNotFound, errors.New("Not found environment variable 'XPUB'"))
		}

		encryptedPath, isPresent := os.LookupEnv("PATH")
		if !isPresent {
			return serverError(http.StatusNotFound, errors.New("Not found environment variable 'PATH'"))
		}

		bucket, isPresent := os.LookupEnv("BUCKET_NAME")
		if !isPresent {
			return serverError(http.StatusNotFound, errors.New("Not found environment variable 'BUCKET_NAME'"))
		}

		indexFile, isPresent := os.LookupEnv("INDEX_FILE_NAME")
		if !isPresent {
			return serverError(http.StatusNotFound, errors.New("Not found environment variable 'BUCKET_NAME'"))
		}

		// decrypt xPub and path
		xPubBytes, err := decrypt(encryptedXPub, "Decrypting xPub")
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		pathBytes, err := decrypt(encryptedPath, "Decrypting path")
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		xPub := string(xPubBytes)
		path := string(pathBytes)

		// get index
		s3 := &s3wrapper.S3Wrapper{}
		err = s3.New(Region)
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		index, err := getAddressIndex(s3, bucket, indexFile)
		if err != nil {
			return serverError(http.StatusInternalServerError, err)
		}

		// generate new address
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

// decrypt data using KMS client from just a session.
// return plaintext in byte array
func decrypt(encrypted string, context string) ([]byte, error) {
	kmsClient := kms.New(session.New())
	decodedBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	input := &kms.DecryptInput{
		CiphertextBlob: decodedBytes,
	}

	response, err := kmsClient.Decrypt(input)
	if err != nil {
		return nil, err
	}

	return response.Plaintext[:], nil
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

	responseError := &lambdaError{
		Status:  status,
		Message: err.Error(),
	}
	jsonResponseError, err := json.Marshal(responseError)
	if err != nil {
		return basicError, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(jsonResponseError),
	}, nil
}
