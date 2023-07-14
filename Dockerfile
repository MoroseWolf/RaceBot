FROM golang:1.20-alpine

RUN mkdir /app

ADD . /app/
WORKDIR /app

ENV RACEVK_BOT="Token"

RUN go mod download
RUN go build -o racebot_vk ./main
CMD ["./racebot_vk"]