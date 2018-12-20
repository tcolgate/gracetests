# BUILD Stage
FROM python:3.6-stretch AS build

ARG PIP_CONF

RUN mkdir -p /root/.pip
RUN echo ${PIP_CONF} | base64 --decode >> /root/.pip/pip.conf
COPY requirements.txt /
RUN pip install --upgrade pip-tools
RUN pip-sync /requirements.txt

# Final Image
FROM python:3.6-stretch

COPY --from=build /usr/local/lib/python3.6 /usr/local/lib/python3.6

RUN mkdir -p /app
WORKDIR /app

COPY graceful /app/graceful
COPY setup.py /app/setup.py

ENV PYTHONPATH "/app"
ENV PROJECT graceful
ENTRYPOINT ["/usr/local/bin/python3", "/app/graceful/main.py"]
