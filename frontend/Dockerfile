FROM ubuntu:24.04 AS sdk
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git unzip xz-utils libglu1-mesa && rm -rf /var/lib/apt/lists/*
ARG FLUTTER_VERSION=3.44.0
RUN git clone --depth 1 --branch ${FLUTTER_VERSION} https://github.com/flutter/flutter.git /opt/flutter
ENV PATH="/opt/flutter/bin:/opt/flutter/bin/cache/dart-sdk/bin:${PATH}"
RUN flutter config --no-analytics && flutter precache --web
WORKDIR /app

FROM sdk AS dependencies
COPY pubspec.yaml pubspec.lock* ./
RUN flutter pub get
COPY . .

FROM dependencies AS test
CMD ["sh", "-c", "flutter analyze && flutter test --reporter expanded"]

FROM dependencies AS build
RUN flutter build web --release --no-web-resources-cdn

FROM nginx:1.28-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/build/web /usr/share/nginx/html
EXPOSE 80
