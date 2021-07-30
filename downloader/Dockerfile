FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y \
  && yum --disableplugin=subscription-manager -y install \
  httpd \
  && yum --disableplugin=subscription-manager clean all

RUN sed -i 's/Listen 80/Listen 8080/' /etc/httpd/conf/httpd.conf \
  && chgrp -R 0 /var/log/httpd /var/run/httpd \
  && chmod -R g=u /var/log/httpd /var/run/httpd

EXPOSE 8080

USER 1001

CMD httpd -D FOREGROUND