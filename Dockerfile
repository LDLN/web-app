# #  Copyright 2014-2015 LDLN
# 
#  This file is part of LDLN Base Station.
# 
#  LDLN Base Station is free software: you can redistribute it and/or modify
#  it under the terms of the GNU General Public License as published by
#  the Free Software Foundation, either version 3 of the License, or
#  any later version.
#
#  LDLN Base Station is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
# 
#  You should have received a copy of the GNU General Public License
#  along with LDLN Base Station.  If not, see <http://www.gnu.org/licenses/>.
# 

# Use the official go docker image built on debian.
FROM golang:1.7

# Grab the source code and add it to the workspace.
#ADD . /go/src/github.com/ldln/web-app
RUN apt-get install gcc libc6-dev git mercurial bzr -y 

# Install revel and the revel CLI.
RUN go get github.com/revel/revel
RUN go get github.com/revel/cmd/revel
RUN go get golang.org/x/crypto/bcrypt
RUN go get github.com/nu7hatch/gouuid
RUN go get labix.org/v2/mgo
RUN go get golang.org/x/net/websocket
RUN go get github.com/gorilla/websocket
RUN go get github.com/hashicorp/mdns
#RUN go get github.com/revel/modules/static

#RUN go get -d github.com/ldln/core
#RUN go get github.com/ldln/web-app
RUN go get github.com/ldln/websocket-server
RUN go get github.com/ldln/websocket-client
RUN go get github.com/ldln/serial-server

RUN revel build github.com/ldln/web-app

# Use the revel CLI to start up our application.
#ENTRYPOINT sudo $GOPATH/bin/web-app/run.sh


# Open up the port where the app is running.
EXPOSE 80
EXPOSE 8080
