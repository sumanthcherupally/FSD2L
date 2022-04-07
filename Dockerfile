FROM ubuntu:latest

MAINTAINER Srinivas < bogasrinu777@gmail.com >

WORKDIR /app

COPY ./main bootstrapNodes.txt ./

CMD ["./main"]
