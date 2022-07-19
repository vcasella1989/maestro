#bin/bash

apt-get update
mkdir /opt/maestro
tar -xvf maestro.tar --directory /opt/

export DEBIAN_FRONTEND=noninteractive
cd /opt/maestro/bin
./maestro