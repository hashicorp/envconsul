Dockerfile
=========
The Dockerfile was created to embed `envetcd` in each docker container and run it directly in order to wrap the Go executable for the services. 

Installation
------------
The contents of the Dockerfile in this directory should be prepended into the Dockerfiles of each service. Building the service docker image will add the `envetcd` binary into /envetcd directory. A RUN command will be used to automatically execute `envetcd` to get the Services' environment variables. However another script or program is needed to write the env variables into the container. 


```shell

#modify base image for each service
FROM ubuntu:14.04 

# Add envetcd into Docker Image at /envetcd/
ADD envetcd /envetcd/
WORKDIR /envetcd

RUN ./envtcd --system $SYSTEM --service $SERVICE -c env > $TMPFILE_ENVETCD
RUN {script that will write the environment variables from the TMPFILE_ENVETCD}
RUN rm -f $TMPFILE_ENVETCD

```
