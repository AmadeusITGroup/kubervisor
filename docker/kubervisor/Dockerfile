FROM busybox

COPY ./kubervisor /kubervisor

# Promethes endpoint
EXPOSE 9091

ENTRYPOINT [ "/kubervisor" ]
