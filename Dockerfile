FROM golang:1.12
ENV workdir /build
WORKDIR $workdir
COPY . .

RUN go install -v .

CMD ["crcards"]
