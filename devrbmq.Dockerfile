FROM rabbitmq:alpine

RUN rabbitmq-plugins enable --offline rabbitmq_management

EXPOSE 15672:15672
