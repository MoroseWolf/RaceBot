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

ENV RACEVK_BOT="vk1.a.hOzIGOd4fx42vEE3H-R8MR5bTnPGD-6VOhE2jc_7o9S2S44bAaAzA-X8ssaNJUuDkVk2OAi8349H1ZaQORNxCj2Cyaghor6KHF9RUnOxErLl0yhVXdRebcGsn4pr1WhKFwJGdHzSexmCpQd8hyJeGhwyU8YZpQWUzbKHV20LoGy5OUn1XY0gwk9AExD7w073iJdCDV42Y83KZrUYt2GkHQ"

RUN go mod download
RUN go build -o racebot_vk ./main/
CMD ["./racebot_vk"]