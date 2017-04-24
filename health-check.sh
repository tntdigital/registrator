#!/usr/bin/env sh
set -e

# Check if registrator is connected to the Docker socket
if [ ! $(netstat -x | grep '/var/run/docker.sock' | grep 'CONNECTED' | wc -l) -gt 0 ]; then
  echo 'Registrator is not connected to Docker';
  exit 1
fi
echo 'Registrator is connected to Docker'


# Check if registrator is connected to Consul
if [ ! $(netstat -tn | grep ${CONSUL_HTTP_ADDR} | grep 'ESTABLISHED' | wc -l) -gt 0 ]; then
  echo 'Registrator is not connected to Consul';
  exit 1
fi
echo 'Registrator is connected to Consul'

echo 'Registrator is up and running'
