#!/bin/bash

ROOT_DIR=$(cd $(dirname $0)/..; pwd)
DB_DIR="$ROOT_DIR/db"
BENCH_DIR="$ROOT_DIR/bench"

export MYSQL_PWD=isucon
mysql -uisucon -e "DROP DATABASE IF EXISTS torb; CREATE DATABASE torb;"
mysql -uisucon torb < "$DB_DIR/schema.sql"

if [ ! -f "$DB_DIR/isucon8q-initial-dataset.sql.gz" ]; then
  echo "Run the following command beforehand." 1>&2
  echo "$ ( cd \"$BENCH_DIR\" && bin/gen-initial-dataset )" 1>&2
  exit 1
fi

gzip -dc "$DB_DIR/isucon8q-initial-dataset.sql.gz" | mysql -uisucon torb

mysql -uisucon torb -e "ALTER TABLE sheets ADD UNIQUE rank_num_uniq (\`rank\`, num)"&
mysql -uisucon torb -e "ALTER TABLE reservations ADD INDEX user_id (user_id)"&
mysql -uisucon torb -e "ALTER TABLE reservations ADD INDEX event_cancel (event_id, canceled_at)"&
mysql -uisucon torb -e "ALTER TABLE reservations ADD UNIQUE uniq3 (event_id, sheet_id, canceled_at)"&


mysql -uisucon torb -e 'update reservations set event_price = (select price from events where id = reservations.event_id)'
