FROM busybox

COPY ./customanomalydetector /customanomalydetector

# Promethes endpoint
EXPOSE 8080

ENTRYPOINT [ "/customanomalydetector" ]
