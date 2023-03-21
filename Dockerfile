# docker run --rm -ti \
#	--name webtester
#	-p 80:80
#	cuotos/webtester
#

FROM golang as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o /webtester .

FROM alpine
COPY --from=builder /webtester /webtester
EXPOSE 5117
CMD /webtester