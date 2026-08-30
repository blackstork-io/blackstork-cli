# syntax=docker/dockerfile:1

FROM alpine:3.19
ENTRYPOINT [ "/blackstork-cli" ]
CMD [ "--help" ]
COPY blackstork-cli /blackstork-cli
