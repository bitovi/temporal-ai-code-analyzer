#!/bin/bash

docker compose down worker && docker compose up --build -d worker
