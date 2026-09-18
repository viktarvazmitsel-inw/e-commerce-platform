FROM php:8.5-fpm-alpine3.23

WORKDIR /var/www/html

ARG UID=1000
ARG GID=1000

RUN apk add --no-cache shadow \
    && usermod -u ${UID} www-data \
    && groupmod -g ${GID} www-data

RUN apk add --no-cache \
    postgresql-dev \
    libpng-dev \
    libjpeg-turbo-dev \
    libzip-dev \
    zip \
    unzip \
    curl \
    git

RUN docker-php-ext-install pdo pdo_pgsql pgsql zip bcmath gd

RUN apk add --no-cache --virtual .build-deps $PHPIZE_DEPS \
    && pecl install redis \
    && docker-php-ext-enable redis \
    && apk del .build-deps

COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

RUN chown -R www-data:www-data /var/www/html

CMD ["php-fpm"]
