FROM golang:1.18-bullseye

RUN apt-get update && apt-get install -y libvips-dev

WORKDIR /app
COPY . /app/

RUN make linux

EXPOSE 6456

CMD ["/app/dist/linux_amd64/e-mage","serve"]
