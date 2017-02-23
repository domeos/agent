FROM docker:1.8

MAINTAINER domeos

# Prepare

RUN apk add --no-cache bash supervisor


RUN mkdir -p /opt/falcon-agent/logs && cd /opt/falcon-agent

# Set work-dir

WORKDIR /opt/falcon-agent

# Copy falcon-agent files

COPY falcon-agent ./falcon-agent
COPY control ./control
COPY cfg.template.json ./cfg.template.json
COPY dockerConnector ./dockerConnector

# Copy agent ini

COPY agent-supervisor.ini  /etc/supervisor.d/agent-supervisor.ini
COPY dockerConnector-supervisor.ini /etc/supervisor.d/dockerConnector.ini
 
# Copy dockerize

COPY dockerize  /usr/local/bin/dockerize

# adjust timezone

COPY Shanghai /etc/localtime

# Start service

CMD dockerize -template ./cfg.template.json:./cfg.json supervisord -n
