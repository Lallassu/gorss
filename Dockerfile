# docker run -it wfnintr/gorss
# docker run -v (pwd)/gorss.conf:/root/gorss/gorss.conf -it wfnintr/gorss
# docker run -v (pwd)/gorss.conf:/root/gorss/gorss.conf -it wfnintr/gorss -theme themes/irssi.theme -db mydb2.db
from alpine:latest
LABEL maintainer="wfnintr@null.net"

RUN wget -qO - https://github.com/Lallassu/gorss/releases/latest/download/gorss_linux.tar.gz | tar xzf - -C /root

WORKDIR /root/gorss
ENTRYPOINT ["./gorss_linux"]
CMD ["-config", "gorss.conf", "-theme", "default.theme", "-db", "mydb.db"]
