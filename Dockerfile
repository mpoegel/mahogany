FROM scratch
COPY mahogany /
ENTRYPOINT [ "/mahogany" ]
COPY static /etc/mahogany/static
ENV STATIC_DIR=/etc/mahogany/static
