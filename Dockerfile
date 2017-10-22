FROM scratch

MAINTAINER Guilherme Silveira <xguiga@gmail.com>

COPY aws-deploy /

EXPOSE 8080

CMD ["/aws-deploy"]