FROM 033441544097.dkr.ecr.us-east-1.amazonaws.com/craft-base

MAINTAINER David Rivas david@momentlabs.io


# This is what the parent expects.
ADD craft-config_linux_amd64 /usr/local/bin/craft-config
RUN chmod a+rx /usr/local/bin/craft-config

WORKDIR /data

ENTRYPOINT ["/usr/local/bin/craft-config", "version"]
