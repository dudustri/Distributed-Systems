#!/bin/bash

# Build Docker images
sudo docker-compose build

# Start Docker containers
sudo docker-compose up -d

# List running containers to check if they are running
sudo docker ps