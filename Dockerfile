FROM alpine:3.3

ADD *.go /factset-reader/

RUN apk add --update bash \
  && apk --update add git bzr \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/factset-reader" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv factset-reader/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && go build \
  && mv factset-reader /factset-reader-app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/factset-reader-app" ]
