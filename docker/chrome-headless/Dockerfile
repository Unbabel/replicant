FROM chromedp/headless-shell:stable

RUN apt update \
&& apt install -y dumb-init procps \
&& apt-get clean \
&& rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY start.sh /
RUN chmod 700 start.sh

ENTRYPOINT ["dumb-init", "--"]

CMD ["/start.sh"]


