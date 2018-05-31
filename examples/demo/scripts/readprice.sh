#!/bin/bash

URI="http://pricer-1a.demo.mk/api/v1/search?origin=PAR&destination=CPH"

echo $URI
while true
do 
    echo "$(curl -s $URI | jq .solutions[].segments[].price.price | cut -b 1-8)"
    sleep 0.1
done