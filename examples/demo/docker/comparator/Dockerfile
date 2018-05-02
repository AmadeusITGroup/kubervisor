FROM alpine
RUN apk add --no-cache curl

ADD comparator /

EXPOSE 8080

ENTRYPOINT [ "/comparator" ]