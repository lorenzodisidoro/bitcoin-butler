# bitcoin-butler
*Hi sir, I'm here to help you.*

*I can generate external bitcoin addresses from your `xPub` keys on public request.*

[<img src="bitcoin_accepted_here.png" width="150" />](https://wszewejuph.execute-api.eu-west-1.amazonaws.com/stage/address)

## Prerequisite
- [Go 1.11](https://golang.org/) or newer 
- [Dep 0.5+](https://github.com/golang/dep/blob/master/README.md)
- AWS CLI installed and configured

## Installation
Clone this repository

```sh
git clone https://github.com/lorenzodisidoro/bitcoin-butler.git
```

and install dependencies by moving into the `bitcoin-butler` directory and running `install` script as a follow
```sh
cd bitcoin-butler && bash ./scripts/install
```

## Build
To build on Linux using `build` script running the following command
```sh
bash ./scripts/build linux amd64
```

or for others OS refer to [a list of valid GOOS and GOARCH](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63).
Build script generate `bitcoin-butler-lambda-linux.zip` zip file to use to deploy the lambda function to AWS.

### Lambda environment
Runtime environment for the lambda function:
- `NETWORK` is bitcoin network
- `XPUB` is extended public key encrypted
- `PATH` is account BIP32 derivation path (eg. `m/44/0/1/0`) encrypted
- `BUCKET_NAME` as the name of bucket to use
- `INDEX_FILE_NAME` as the name of file used to save index

### Lambda policy
You will give at your lambda function the basic permissions required, you can use `resource/aws/lambda_policy.json` and `resource/aws/lambda_role.json` documents.

