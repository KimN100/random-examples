#!/bin/bash

if [[ ! -f "$1" ]]
then
	echo "ERROR: missing file"
	exit
fi

if [[ ! -f "$2" ]]
then
	echo "ERROR: missing blockchain"
	exit
fi

SIZE=`wc -c "$1" | tr -s ' ' | cut -d' ' -f2`
SUM=`cat "$2" "$1" | md5`
SIGNED=`echo $SUM | tr '0123456789abcdef' '89abcdef01234567'`

printf "BLOC%04x\n" $SIZE >> $2
cat "$1" >> "$2"
echo $SIGNED >> $2
