#!/bin/bash
# Конфигурация БД
export PGUSER="validator"
export PGPASSWORD="val1dat0r"
export PGHOST="localhost"
export PGPORT="5432"
export PGDATABASE="project-sem-1"

# Установка зависимостей
echo "Installing Go dependencies..."
go mod download
go mod tidy

# Проверка и инициализация БД
echo "Checking database setup..."

TABLE_EXISTS=$(psql -tAc "SELECT EXISTS (
    SELECT FROM information_schema.tables
    WHERE table_name = 'prices'
)")

if [ "$TABLE_EXISTS" = "t" ]; then
    echo "Table 'prices' already exists"
else
    echo "Creating 'prices' table..."
    psql <<-EOSQL
        CREATE TABLE prices (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            category TEXT NOT NULL,
            price NUMERIC(10,2) NOT NULL,
            create_date TIMESTAMP NOT NULL
        );
        GRANT ALL PRIVILEGES ON TABLE prices TO validator;
        GRANT USAGE, SELECT ON SEQUENCE prices_id_seq TO validator;
EOSQL
    echo "Table 'prices' created successfully"
fi

echo "Database ready!"

##!/bin/bash
#set -e
#
#export PGPASSWORD="val1dat0r"
#
## Установка зависимостей
#go mod download
#
## Пересоздание таблицы
#psql -h localhost -p 5432 -U validator -d project-sem-1 <<-EOSQL
#	DROP TABLE IF EXISTS prices;
#	CREATE TABLE prices (
#			id SERIAL PRIMARY KEY,
#			name TEXT,
#			category TEXT,
#			price NUMERIC,
#			create_date TIMESTAMP
#		);
#	GRANT ALL PRIVILEGES ON TABLE prices TO validator;
#EOSQL
#
#echo "Database prepared successfully!"