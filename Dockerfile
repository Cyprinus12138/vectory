FROM golang:1.20
ARG GIT_TOKEN=""
WORKDIR /app
COPY . ./
RUN \
    if [ -n "${GIT_TOKEN}" ] ; \
    then git config --global url."https://$GIT_TOKEN@github.com/".insteadOf "https://github.com/" ; \
    else \
    echo "no token provided" ; \
    fi
RUN make setup && make build
ENTRYPOINT ["./bin/server/main"]
