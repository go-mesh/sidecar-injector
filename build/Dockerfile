FROM ubuntu:16.04 

ADD conf .
ADD sidecar-injector /sidecar-injector
RUN mkdir -p log
ENTRYPOINT ["./sidecar-injector"]
