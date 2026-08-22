FROM nginx:1.27-alpine

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache wget tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY www /usr/share/nginx/html

EXPOSE 80
