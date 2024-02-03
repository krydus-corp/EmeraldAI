#!/bin/bash
set -e

AUTH_DB=admin
CURRENTDATETIME=`date +"%Y-%m-%d"`
OUT_DIR=exports/$CURRENTDATETIME

echo $OUT_DIR
display_usage() {
  cli_name=${0##*/}
  echo "
$cli_name
MongoDB Import-Export
Usage: $cli_name [connection-uri] [database] [comma-delimited-list-of-collections]
Commands:
  import    Import collections at connection-uri
  export    Export colelction at connection-uri
  help      Display help
"
  exit 1
}


# If less than three arguments supplied, display usage
if [  $# -le 0 ]
then
    display_usage
    exit 1
fi

CMD=
URI=$2
DB=$3
COLLECTIONS=($4)

case "$1" in
 -e|--export)
  echo "Exporting ${#COLLECTIONS[@]} collections -> ${COLLECTIONS[@]}"
  for i in ${COLLECTIONS[@]}
  do
    mongoexport --uri=$URI --db $DB --authenticationDatabase $AUTH_DB --collection=$i  --out=$OUT_DIR/$i.json
  done
  ;;

 -i|--import)
  echo "Importing ${#COLLECTIONS[@]} collections -> ${COLLECTIONS[@]}"
  for i in ${COLLECTIONS[@]}
  do
    mongoimport --uri=$URI --db $DB --authenticationDatabase $AUTH_DB --collection=$i  --file=$OUT_DIR/$i.json
  done
  ;;

 *)
  # else
  echo display_usage
  ;;
esac
