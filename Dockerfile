# docker run --rm -ti \
#	--name webtester
#	-p 80:80
#	cuotos/webtester
#

FROM nginx:alpine

RUN echo -e "#!/bin/sh\nhostname >/usr/share/nginx/html/index.html\nnginx -g \"daemon off;\"" >>/start.sh && chmod +x /start.sh && touch /usr/share/nginx/html/healthz

CMD ["/start.sh"]
