#!/bin/sh

while ! nc -z $DB_HOST $DB_PORT; do
  sleep 1
done

echo "PostgreSQL is up - starting the app..."
exec "$@"
