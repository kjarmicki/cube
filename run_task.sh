#!/bin/bash

curl -v --request POST \
--header 'Content-Type: application/json' \
--data @task.json \
localhost:3030/tasks
