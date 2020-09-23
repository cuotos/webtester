#!/bin/bash

hostname >/usr/share/nginx/html/index.html
echo $TEXT >> /usr/share/nginx/html/index.html
nginx -g "daemon off;" 
