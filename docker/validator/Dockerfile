# commons-ip CSIP validator (see docs/decisions/0005-dockerized-validation-and-html-reporting.md).
# The jar is pinned by version AND sha256 so a re-tagged release can't silently
# change what "valid" means; bump both together, deliberately.
FROM eclipse-temurin:21-jre-alpine

ARG COMMONS_IP_VERSION=2.11.2
ARG COMMONS_IP_SHA256=e43bd94bb37a6f04a192b160010fc7e1e4cc1423f9d41fd04f61d8303ec3648e

# chmod: ADD from a URL creates the file 0600 root-only, but validate.sh runs
# the container as the host user so report files land host-owned.
ADD https://github.com/keeps/commons-ip/releases/download/${COMMONS_IP_VERSION}/commons-ip2-cli-${COMMONS_IP_VERSION}.jar /opt/commons-ip.jar
RUN echo "${COMMONS_IP_SHA256}  /opt/commons-ip.jar" | sha256sum -c - && chmod 0644 /opt/commons-ip.jar

ENTRYPOINT ["java", "-jar", "/opt/commons-ip.jar"]
