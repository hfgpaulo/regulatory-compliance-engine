# Estágio de dependências: instala só as de produção (sem PHPUnit/PHPStan) e
# gera o autoload otimizado depois de copiar o código.
FROM composer:2 AS deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-interaction --no-progress --prefer-dist --no-autoloader
COPY src ./src
RUN composer dump-autoload --no-dev --optimize

# Estágio final: Apache + mod_php, a combinação típica de um legado PHP.
# O DocumentRoot aponta para public/, então src/ e vendor/ não ficam expostos.
FROM php:8.3-apache
ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN docker-php-ext-install pdo_mysql \
    && sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
    && a2enmod rewrite \
    && echo "ServerName localhost" > /etc/apache2/conf-available/servername.conf \
    && a2enconf servername
WORKDIR /var/www/html
COPY --from=deps /app/vendor ./vendor
COPY src ./src
COPY public ./public
COPY bin ./bin
EXPOSE 80
# Migração idempotente do banco legado antes de o Apache aceitar tráfego;
# se falhar, o container não sobe (fail-fast, como o motor sem Mongo).
CMD ["sh", "-c", "php bin/migrate.php && exec apache2-foreground"]
