#!/bin/bash

hostname >/usr/share/nginx/html/index.html
nginx -g "daemon off;" 
