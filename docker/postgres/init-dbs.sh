#!/bin/bash

for cred in $INIT_DBS; do
    name=`echo $cred | cut -d: -f 1`
    pass=`echo $cred | cut -d: -f 2`
    dbname="${name}db"

    psql -c "CREATE ROLE $name WITH LOGIN PASSWORD '$pass';"
    psql -c "CREATE DATABASE $dbname WITH OWNER $name;"
    psql -U "$name" -d "$dbname" < "/usr/local/share/db-init/${name}.sql"
done
