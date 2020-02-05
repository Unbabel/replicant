#!/bin/bash

# nasty hack to workaround the ever growing utility process
while true; do
	sleep 60
	pid=$(ps -ef|grep "type=utility"|grep -v grep|awk '{print $2}')
	[[ -n $pid ]] && kill -HUP $pid
done &

exec /headless-shell/headless-shell \
--headless \
--no-zygote \
--no-sandbox \
--disable-gpu \
--disable-software-rasterizer \
--disable-dev-shm-usage \
--remote-debugging-address=0.0.0.0 \
--remote-debugging-port=9222 \
--incognito \
--disable-shared-workers \
--disable-remote-fonts