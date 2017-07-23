# This Dockerfile builds on golang:alpine by building Blackice from source
# using the current working directory.
FROM golang:alpine
MAINTAINER "Alvaro Mongil <hello@alvaromongil.com>"
RUN apk add --no-cache git

# Copy the local package files to the container's workspace.
ADD . $GOPATH/src/github.com/amongil/blackice

# Get the dependencies
RUN go get github.com/aws/aws-sdk-go/aws
RUN go get github.com/aws/aws-sdk-go/aws/awserr
RUN go get github.com/aws/aws-sdk-go/aws/session
RUN go get github.com/aws/aws-sdk-go/service/ec2
RUN go get github.com/julienschmidt/httprouter
RUN go get github.com/spf13/cobra

# Build the Blackice tool inside the container.
RUN go install github.com/amongil/blackice/blackice

# Run the start command by default when the container starts.
ENTRYPOINT $GOPATH/bin/blackice start

# Document that the service listens on port 8080.
EXPOSE 8080