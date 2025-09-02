FROM maven:3.9.9-eclipse-temurin-17 AS sdk
WORKDIR /sdk
RUN git clone https://github.com/dapr/java-sdk.git . \
    && git submodule update --init --recursive \
    && mvn -B -q -DskipTests -T1C clean install

FROM maven:3.9.9-eclipse-temurin-17 AS build
WORKDIR /workspace
COPY --from=sdk /root/.m2 /root/.m2
COPY pom.xml ./
COPY src ./src
# Auto-detect the built dapr-sdk version from the local Maven repo copied from the SDK stage
RUN DAPR_SDK_VERSION=1.17.0-SNAPSHOT \
    && echo "Using Dapr SDK version: ${DAPR_SDK_VERSION}" \
    && mvn -q -DskipTests -Ddapr.sdk.version=${DAPR_SDK_VERSION} -Ddapr.sdk.workflows.version=${DAPR_SDK_VERSION} package

FROM eclipse-temurin:17-jre
WORKDIR /app
ENV APP_PORT=6008
COPY --from=build /workspace/target/*.jar app.jar
EXPOSE 6008
ENTRYPOINT ["java","-jar","/app/app.jar"]
