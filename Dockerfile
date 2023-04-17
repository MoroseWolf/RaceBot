FROM golang:latest
##
##WORKDIR /RaceBot_VK
###
##COPY go.mod ./
##COPY go.sum ./
##RUN go mod download

##COPY ./main ./
##RUN go version

##RUN go build -o /racebot_vk
##CMD ["racebot_vk", "./main/main.go"]

RUN mkdir /app

ADD . /app/
WORKDIR /app

##ENV RACEBOT_VK = "Token"

RUN go mod download
RUN go build -o racebot_vk ./main/
CMD ["./racebot_vk"]
